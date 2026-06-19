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
	"github.com/evg4b/uncors/pkg/urlt"
)

const nanosecondsPerMillisecond = 1e6

var secureHeaderNames = map[string]bool{
	"Cookie":              true,
	"Set-Cookie":          true,
	"Authorization":       true,
	"Www-Authenticate":    true, // Go canonicalises header names
	"Proxy-Authorization": true,
	"Proxy-Authenticate":  true,
}

type Middleware struct {
	writer               *Writer
	captureSecureHeaders bool
}

func NewMiddleware(opts ...MiddlewareOption) *Middleware {
	m := helpers.ApplyOptions(&Middleware{}, opts)
	if m.writer == nil {
		panic("har: NewMiddleware requires WithWriter option")
	}

	return m
}

func (m *Middleware) Wrap(next contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(writer contracts.ResponseWriter, req *contracts.Request) error {
		start := time.Now()

		var reqBodySize int64

		if req.Body != nil && req.Body != http.NoBody {
			var buf strings.Builder

			n, _ := io.Copy(&buf, req.Body)
			reqBodySize = n
			_ = req.Body.Close()

			req.Body = io.NopCloser(strings.NewReader(buf.String()))
		}

		writer.EnableBodyCapture()

		err := next.ServeHTTP(writer, req)

		elapsed := time.Since(start)
		entry := m.buildEntry(req, writer.Captured(), start, elapsed, reqBodySize)
		m.writer.AddEntry(entry)

		return err
	})
}

func (m *Middleware) buildEntry(
	req *http.Request,
	capture contracts.ResponseCapture,
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
		QueryString: queryToNameValues(urlt.URL_Query(req.URL)),
		Cookies:     cookies,
		HeadersSize: -1,
		BodySize:    bodySize,
	}
}

func (m *Middleware) buildResponse(capture contracts.ResponseCapture) Response {
	mimeType := capture.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	var cookies []Cookie

	if m.captureSecureHeaders {
		cookies = cookiesToHAR(extractResponseCookies(capture.Header))
	}

	rawBody := capture.Body
	content := buildContent(rawBody, capture.Header.Get("Content-Encoding"), mimeType)

	return Response{
		Status:      capture.StatusCode,
		StatusText:  http.StatusText(capture.StatusCode),
		HTTPVersion: "HTTP/1.1",
		Headers:     m.headersToNameValues(capture.Header),
		Cookies:     cookies,
		Content:     content,
		RedirectURL: capture.Header.Get("Location"),
		HeadersSize: -1,
		BodySize:    int64(len(rawBody)),
	}
}

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
	resp := &http.Response{Header: h}

	return resp.Cookies()
}
