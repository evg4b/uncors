package har

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
)

// nanosecondsPerMillisecond is used to convert nanoseconds to milliseconds.
const nanosecondsPerMillisecond = 1e6

// secureHeaderNames is the set of HTTP headers that may carry credentials or
// session data. They are stripped from HAR entries by default and are only
// included when captureSecureHeaders is true.
//
// The list covers RFC-defined authentication and cookie headers:
//   - Cookie / Set-Cookie  — session identifiers
//   - Authorization        — Bearer tokens, Basic credentials
//   - WWW-Authenticate     — server auth challenges (reveals scheme/realm)
//   - Proxy-Authorization  — proxy credentials
//   - Proxy-Authenticate   — proxy auth challenges
var secureHeaderNames = map[string]bool{
	"Cookie":              true,
	"Set-Cookie":          true,
	"Authorization":       true,
	"Www-Authenticate":    true, // Go canonicalises header names
	"Proxy-Authorization": true,
	"Proxy-Authenticate":  true,
}

// Middleware captures every request/response pair and enqueues a HAR
// entry to the async Writer. The handler chain is never blocked.
type Middleware struct {
	writer               *Writer
	captureSecureHeaders bool
}

// NewMiddleware creates a Middleware backed by the given Writer.
func NewMiddleware(opts ...MiddlewareOption) *Middleware {
	return helpers.ApplyOptions(&Middleware{}, opts)
}

// Wrap returns a Handler that records the transaction before passing
// control to next.
func (m *Middleware) Wrap(next contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(w contracts.ResponseWriter, req *contracts.Request) {
		start := time.Now()

		capture := newCaptureWriter(w)

		var reqBodySize int64

		if req.Body != nil && req.Body != http.NoBody {
			var buf strings.Builder

			n, _ := io.Copy(&buf, req.Body)
			reqBodySize = n
			_ = req.Body.Close()

			// Restore body so the next handler can read it.
			req.Body = io.NopCloser(strings.NewReader(buf.String()))
		}

		next.ServeHTTP(capture, req)

		elapsed := time.Since(start)

		entry := m.buildEntry(req, capture, start, elapsed, reqBodySize)

		m.writer.AddEntry(entry)
	})
}

func (m *Middleware) buildEntry(
	req *http.Request,
	capture *captureWriter,
	start time.Time,
	elapsed time.Duration,
	reqBodySize int64,
) Entry {
	elapsedMS := float64(elapsed.Nanoseconds()) / nanosecondsPerMillisecond

	return Entry{
		StartedDateTime: start,
		Time:            elapsedMS,
		Request:         m.buildRequest(req, reqBodySize),
		Response:        m.buildResponse(capture),
		Timings: Timings{
			Send:    0,
			Wait:    elapsedMS,
			Receive: 0,
		},
	}
}

func (m *Middleware) buildRequest(req *http.Request, bodySize int64) Request {
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}

	fullURL := fmt.Sprintf("%s://%s%s", scheme, req.Host, req.RequestURI)

	var cookies []Cookie

	if m.captureSecureHeaders {
		cookies = cookiesToHAR(req.Cookies())
	}

	return Request{
		Method:      req.Method,
		URL:         fullURL,
		HTTPVersion: req.Proto,
		Headers:     m.headersToNameValues(req.Header),
		QueryString: queryToNameValues(req.URL.Query()),
		Cookies:     cookies,
		HeadersSize: -1,
		BodySize:    bodySize,
	}
}

func (m *Middleware) buildResponse(capture *captureWriter) Response {
	mimeType := capture.Header().Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	var cookies []Cookie

	if m.captureSecureHeaders {
		cookies = cookiesToHAR(extractResponseCookies(capture.Header()))
	}

	rawBody := capture.body()
	content := buildContent(rawBody, capture.Header().Get("Content-Encoding"), mimeType)

	return Response{
		Status:      capture.code,
		StatusText:  http.StatusText(capture.code),
		HTTPVersion: "HTTP/1.1",
		Headers:     m.headersToNameValues(capture.Header()),
		Cookies:     cookies,
		Content:     content,
		RedirectURL: capture.Header().Get("Location"),
		HeadersSize: -1,
		BodySize:    int64(len(rawBody)),
	}
}

// headersToNameValues converts an http.Header to a slice of NameValue pairs.
// When captureSecureHeaders is false, all headers in secureHeaderNames are
// excluded to avoid persisting credentials in the HAR file.
func (m *Middleware) headersToNameValues(h http.Header) []NameValue {
	result := make([]NameValue, 0, len(h))

	for name, values := range h {
		if !m.captureSecureHeaders && secureHeaderNames[name] {
			continue
		}

		for _, v := range values {
			result = append(result, NameValue{Name: name, Value: v})
		}
	}

	return result
}

func queryToNameValues(q url.Values) []NameValue {
	result := make([]NameValue, 0, len(q))

	for k, vals := range q {
		for _, v := range vals {
			result = append(result, NameValue{Name: k, Value: v})
		}
	}

	return result
}

func cookiesToHAR(cookies []*http.Cookie) []Cookie {
	result := make([]Cookie, 0, len(cookies))

	for _, c := range cookies {
		result = append(result, Cookie{Name: c.Name, Value: c.Value})
	}

	return result
}

func extractResponseCookies(h http.Header) []*http.Cookie {
	// Parse Set-Cookie headers via a synthetic response.
	resp := &http.Response{Header: h}

	return resp.Cookies()
}
