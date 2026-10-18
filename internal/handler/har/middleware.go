package har

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
	"unicode/utf8"

	harlib "github.com/evg4b/uncors/pkg/har"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/pkg/urlt"
	"go.opentelemetry.io/otel/attribute"
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

	helpers.AssertIsDefined(m.writer, "HARMiddleware: Writer is not configured")

	return m
}

func (m *Middleware) ServeHTTP(writer contracts.ResponseWriter, req *contracts.Request, next contracts.Next) error {
	req, span := telemetry.StartRequest(req, "har.record")
	defer span.End()

	start := time.Now()

	var reqBody []byte

	if req.Body != nil && req.Body != http.NoBody {
		body, err := io.ReadAll(req.Body)
		_ = req.Body.Close()

		if err != nil {
			telemetry.Fail(span, err)

			return fmt.Errorf("har: cannot read request body: %w", err)
		}

		reqBody = body
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	reqBodySize := int64(len(reqBody))

	writer.EnableBodyCapture()

	err := next(writer, req)

	elapsed := time.Since(start)
	entry := m.buildEntry(req, writer.Captured(), start, elapsed, reqBody)
	recorded := m.writer.AddEntry(entry)
	telemetry.HAREntry(req.Context(), recorded)

	if !recorded && telemetry.Enabled() {
		telemetry.Warn(req.Context(), "HAR entry dropped: the write queue is full",
			slog.String("file", m.writer.Path()), slog.String("url", helpers.LogURL(req.URL)))
	}

	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("uncors.har.file", m.writer.Path()),
			attribute.Int64("uncors.har.request_body_bytes", reqBodySize),
			attribute.Int64("uncors.har.response_body_bytes", entry.Response.BodySize),
			attribute.Bool("uncors.har.recorded", recorded),
			attribute.Bool("uncors.har.secure_headers", m.captureSecureHeaders),
		)
	}

	return err
}

func (m *Middleware) buildEntry(
	req *http.Request,
	capture contracts.ResponseCapture,
	start time.Time,
	elapsed time.Duration,
	reqBody []byte,
) harlib.Entry {
	elapsedMS := float64(elapsed.Nanoseconds()) / nanosecondsPerMillisecond

	return harlib.Entry{
		StartedDateTime: start,
		Time:            elapsedMS,
		Request:         m.buildRequest(req, reqBody),
		Response:        m.buildResponse(req, capture),
		Timings: harlib.Timings{
			Send:    0,
			Wait:    elapsedMS,
			Receive: 0,
		},
	}
}

func (m *Middleware) buildRequest(req *http.Request, body []byte) harlib.Request {
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}

	fullURL := fmt.Sprintf("%s://%s%s", scheme, req.Host, req.RequestURI)

	var cookies []harlib.Cookie

	if m.captureSecureHeaders {
		cookies = cookiesToHAR(req.Cookies())
	}

	return harlib.Request{
		Method:      req.Method,
		URL:         fullURL,
		HTTPVersion: req.Proto,
		Headers:     m.headersToNameValues(req.Header),
		QueryString: queryToNameValues(urlt.URL_Query(req.URL)),
		Cookies:     cookies,
		HeadersSize: -1,
		BodySize:    int64(len(body)),
		PostData:    buildPostData(body, req.Header.Get("Content-Type")),
	}
}

func buildPostData(body []byte, mimeType string) *harlib.PostData {
	if len(body) == 0 {
		return nil
	}

	postData := &harlib.PostData{MimeType: mimeType}
	if utf8.Valid(body) {
		postData.Text = string(body)
	}

	return postData
}

func (m *Middleware) buildResponse(req *http.Request, capture contracts.ResponseCapture) harlib.Response {
	mimeType := capture.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	var cookies []harlib.Cookie

	if m.captureSecureHeaders {
		cookies = cookiesToHAR(extractResponseCookies(capture.Header))
	}

	rawBody := capture.Body
	content := buildContent(rawBody, capture.Header.Get("Content-Encoding"), mimeType)

	return harlib.Response{
		Status:      capture.StatusCode,
		StatusText:  http.StatusText(capture.StatusCode),
		HTTPVersion: req.Proto,
		Headers:     m.headersToNameValues(capture.Header),
		Cookies:     cookies,
		Content:     content,
		RedirectURL: capture.Header.Get("Location"),
		HeadersSize: -1,
		BodySize:    int64(len(rawBody)),
	}
}

func (m *Middleware) headersToNameValues(h http.Header) []harlib.NameValue {
	result := make([]harlib.NameValue, 0, len(h))

	for name, values := range h {
		if !m.captureSecureHeaders && secureHeaderNames[name] {
			continue
		}

		for _, v := range values {
			result = append(result, harlib.NameValue{Name: name, Value: v})
		}
	}

	return result
}

func queryToNameValues(q url.Values) []harlib.NameValue {
	result := make([]harlib.NameValue, 0, len(q))

	for k, vals := range q {
		for _, v := range vals {
			result = append(result, harlib.NameValue{Name: k, Value: v})
		}
	}

	return result
}

func cookiesToHAR(cookies []*http.Cookie) []harlib.Cookie {
	result := make([]harlib.Cookie, 0, len(cookies))

	for _, c := range cookies {
		result = append(result, harlib.Cookie{Name: c.Name, Value: c.Value})
	}

	return result
}

func extractResponseCookies(h http.Header) []*http.Cookie {
	resp := &http.Response{Header: h}

	return resp.Cookies()
}
