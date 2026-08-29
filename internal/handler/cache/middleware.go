package cache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/urlt"
	"github.com/go-http-utils/headers"
	"github.com/samber/lo"
)

// maxCacheableBodySize bounds the request body the cache is willing to hash.
// A larger request is served without caching rather than buffered.
const maxCacheableBodySize = 1 << 20 // 1 MiB

type Middleware struct {
	cache     contracts.Cache
	methods   []string
	pathGlobs config.CacheGlobs
}

func NewMiddleware(options ...MiddlewareOption) *Middleware {
	middleware := helpers.ApplyOptions(&Middleware{}, options)

	if middleware.cache == nil {
		panic("CacheMiddleware: cache storage is not configured")
	}

	return middleware
}

func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return infra.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) error {
		isCacheable, err := m.isCacheableRequest(request)
		if err != nil {
			return err
		}

		if !isCacheable {
			next.ServeHTTP(writer, request)

			return nil
		}

		m.cacheRequest(writer, request, next)

		return nil
	})
}

func (m *Middleware) cacheRequest(writer http.ResponseWriter, request *http.Request, next http.Handler) {
	cacheKey, keyable := m.extractCacheKey(request)
	if !keyable {
		// A request we cannot identify exactly must not be served from, or
		// stored in, the cache: that is how one POST body ends up answered with
		// another one's response.
		next.ServeHTTP(writer, request)

		return
	}

	if cachedResponse := m.getCachedResponse(cacheKey); cachedResponse != nil {
		m.writeCachedResponse(writer, cachedResponse)

		return
	}

	// The pipeline normally installs a recorder; when it did not, cache through
	// one of our own instead of silently caching nothing.
	capturer, ok := infra.CaptureFrom(writer)
	if !ok {
		recorder := infra.NewResponseRecorder(writer)
		capturer, writer = recorder, recorder
	}

	capturer.EnableBodyCapture(infra.DefaultCaptureLimit)

	next.ServeHTTP(writer, request)

	m.storeResponse(cacheKey, request, capturer.Captured())
}

func (m *Middleware) storeResponse(key string, request *http.Request, capture contracts.ResponseCapture) {
	if !m.isStorable(request, capture) {
		return
	}

	headers := lo.MapToSlice(capture.Header, func(name string, values []string) contracts.CachedHeader {
		return contracts.CachedHeader{
			Name:  name,
			Value: values,
		}
	})

	sort.Slice(headers, func(i, j int) bool {
		return headers[i].Name < headers[j].Name
	})

	m.cache.Set(key, contracts.CachedResponse{
		Code:    capture.StatusCode,
		Body:    capture.Body,
		Headers: headers,
	})
}

func (m *Middleware) writeCachedResponse(writer http.ResponseWriter, cachedResponse *contracts.CachedResponse) {
	header := writer.Header()

	for _, cachedHeader := range cachedResponse.Headers {
		for _, value := range cachedHeader.Value {
			header.Add(cachedHeader.Name, value)
		}
	}

	writer.WriteHeader(cachedResponse.Code)

	if len(cachedResponse.Body) > 0 {
		// The body is the upstream's own response, replayed verbatim.
		//nolint:gosec // G705
		_, err := writer.Write(cachedResponse.Body)
		if err != nil {
			// A client that hung up mid-response is normal.
			slog.Debug("cache: failed to write the cached response", "err", err)
		}
	}
}

// isStorable applies the minimum HTTP caching rules. Without them a response
// carrying Set-Cookie, or one produced for an authenticated request, would be
// replayed to whoever asked next.
func (m *Middleware) isStorable(request *http.Request, capture contracts.ResponseCapture) bool {
	if !infra.Is2xxCode(capture.StatusCode) {
		return false
	}

	// Storing a body we only saw the beginning of would serve a corrupted
	// response on the next hit.
	if capture.Truncated {
		return false
	}

	if capture.Header.Get(headers.SetCookie) != "" {
		return false
	}

	cacheControl := strings.ToLower(capture.Header.Get(headers.CacheControl))
	if strings.Contains(cacheControl, "no-store") || strings.Contains(cacheControl, "private") {
		return false
	}

	if request.Header.Get(headers.Authorization) != "" && !strings.Contains(cacheControl, "public") {
		return false
	}

	return varyIsHandled(capture.Header.Get(headers.Vary))
}

// varyIsHandled reports whether the response's Vary is one the cache key already
// accounts for. Accept-Encoding is part of the key; anything else would need a
// variant index, so those responses are not cached at all rather than being
// served to the wrong client.
func varyIsHandled(vary string) bool {
	for field := range strings.SplitSeq(vary, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}

		if !strings.EqualFold(field, headers.AcceptEncoding) {
			return false
		}
	}

	return true
}

func (m *Middleware) isCacheableRequest(request *http.Request) (bool, error) {
	if !slices.Contains(m.methods, request.Method) {
		return false, nil
	}

	for _, pattern := range m.pathGlobs {
		ok, err := doublestar.PathMatch(pattern, request.URL.Path)
		if err != nil {
			return false, err
		}

		if ok {
			return true, nil
		}
	}

	return false, nil
}

// extractCacheKey identifies a request completely enough to answer it from the
// cache. The host keeps its port (two mappings on different ports are different
// resources), the negotiated encoding is included because it is the one Vary the
// cache honours, and a request body is hashed in — without it, two POSTs to the
// same path are indistinguishable.
//
// It reports false when the request cannot be identified exactly, which is the
// signal to bypass the cache entirely.
func (m *Middleware) extractCacheKey(request *http.Request) (string, bool) {
	bodyHash, ok := m.requestBodyHash(request)
	if !ok {
		return "", false
	}

	values := urlt.URL_Query(request.URL)

	items := make([]string, 0, len(values))
	for key, value := range values {
		sort.Strings(value)

		valuesKey := key + "=" + strings.Join(value, ",")
		items = append(items, valuesKey)
	}

	sort.Strings(items)

	key := fmt.Sprintf("[%s]%s%s?%s|enc=%s",
		request.Method,
		request.URL.Host,
		request.URL.Path,
		strings.Join(items, ";"),
		request.Header.Get(headers.AcceptEncoding))

	if bodyHash != "" {
		key += "|body=" + bodyHash
	}

	return key, true
}

// requestBodyHash reads the request body, hashes it into the key and puts it
// back for the handler. Bodies larger than the cap are not cached at all.
func (m *Middleware) requestBodyHash(request *http.Request) (string, bool) {
	if request.Body == nil || request.Body == http.NoBody {
		return "", true
	}

	body, err := io.ReadAll(io.LimitReader(request.Body, maxCacheableBodySize+1))
	if err != nil {
		return "", false
	}

	_ = request.Body.Close()
	request.Body = io.NopCloser(bytes.NewReader(body))

	if int64(len(body)) > maxCacheableBodySize {
		return "", false
	}

	sum := sha256.Sum256(body)

	return hex.EncodeToString(sum[:]), true
}

func (m *Middleware) getCachedResponse(cacheKey string) *contracts.CachedResponse {
	if cachedResponse, ok := m.cache.Get(cacheKey); ok {
		return &cachedResponse
	}

	return nil
}

type MiddlewareOption = func(*Middleware)

func WithMethods(methods []string) MiddlewareOption {
	return func(m *Middleware) {
		m.methods = methods
	}
}

func WithGlobs(globs config.CacheGlobs) MiddlewareOption {
	return func(m *Middleware) {
		m.pathGlobs = globs
	}
}

func WithCacheStorage(cache contracts.Cache) MiddlewareOption {
	return func(m *Middleware) {
		m.cache = cache
	}
}
