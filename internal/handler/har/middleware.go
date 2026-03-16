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

// Middleware captures every request/response pair and enqueues a HAR
// entry to the async Writer. The handler chain is never blocked.
type Middleware struct {
	writer *Writer
}

// NewMiddleware creates a Middleware backed by the given Writer.
func NewMiddleware(opts ...MiddlewareOption) *Middleware {
	return helpers.ApplyOptions(&Middleware{}, opts)
}

// Wrap returns a Handler that records the transaction before passing
// control to next.
func (m *Middleware) Wrap(next contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(w contracts.ResponseWriter, r *contracts.Request) {
		start := time.Now()

		cw := newCaptureWriter(w)

		var reqBodySize int64

		if r.Body != nil && r.Body != http.NoBody {
			var buf strings.Builder

			n, _ := io.Copy(&buf, r.Body)
			reqBodySize = n
			_ = r.Body.Close()

			// Restore body so the next handler can read it.
			r.Body = io.NopCloser(strings.NewReader(buf.String()))
		}

		next.ServeHTTP(cw, r)

		elapsed := time.Since(start)

		entry := m.buildEntry(r, cw, start, elapsed, reqBodySize)

		m.writer.AddEntry(entry)
	})
}

func (m *Middleware) buildEntry(
	r *http.Request,
	cw *captureWriter,
	start time.Time,
	elapsed time.Duration,
	reqBodySize int64,
) Entry {
	elapsedMS := float64(elapsed.Nanoseconds()) / 1e6

	respBody := cw.body()

	return Entry{
		StartedDateTime: start,
		Time:            elapsedMS,
		Request:         m.buildRequest(r, reqBodySize),
		Response:        m.buildResponse(cw, respBody),
		Timings: Timings{
			Send:    0,
			Wait:    elapsedMS,
			Receive: 0,
		},
	}
}

func (*Middleware) buildRequest(r *http.Request, bodySize int64) Request {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	fullURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI)

	return Request{
		Method:      r.Method,
		URL:         fullURL,
		HTTPVersion: r.Proto,
		Headers:     headersToNameValues(r.Header),
		QueryString: queryToNameValues(r.URL.Query()),
		Cookies:     cookiesToHAR(r.Cookies()),
		HeadersSize: -1,
		BodySize:    bodySize,
	}
}

func (*Middleware) buildResponse(cw *captureWriter, body []byte) Response {
	mimeType := cw.Header().Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	return Response{
		Status:      cw.code,
		StatusText:  http.StatusText(cw.code),
		HTTPVersion: "HTTP/1.1",
		Headers:     headersToNameValues(cw.Header()),
		Cookies:     cookiesToHAR(extractResponseCookies(cw.Header())),
		Content: Content{
			Size:     int64(len(body)),
			MimeType: mimeType,
			Text:     string(body),
		},
		RedirectURL: cw.Header().Get("Location"),
		HeadersSize: -1,
		BodySize:    int64(len(body)),
	}
}

func headersToNameValues(h http.Header) []NameValue {
	result := make([]NameValue, 0, len(h))

	for name, values := range h {
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
