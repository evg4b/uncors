package har

import (
	"net/http"
	"net/url"
	"time"
	"unicode/utf8"

	"github.com/evg4b/uncors/pkg/urlt"
)

const (
	defaultMimeType           = "application/octet-stream"
	nanosecondsPerMillisecond = 1e6
)

var secureHeaderNames = map[string]bool{
	"Cookie":              true,
	"Set-Cookie":          true,
	"Authorization":       true,
	"Www-Authenticate":    true, // Go canonicalises header names
	"Proxy-Authorization": true,
	"Proxy-Authenticate":  true,
}

type options struct {
	secureHeaders bool
}

type Option = func(*options)

func WithSecureHeaders(capture bool) Option {
	return func(o *options) {
		o.secureHeaders = capture
	}
}

func IsSecureHeader(name string) bool {
	return secureHeaderNames[http.CanonicalHeaderKey(name)]
}

func NewEntry(start time.Time, elapsed time.Duration, request Request, response Response) Entry {
	elapsedMS := float64(elapsed.Nanoseconds()) / nanosecondsPerMillisecond

	return Entry{
		StartedDateTime: start,
		Time:            elapsedMS,
		Request:         request,
		Response:        response,
		Timings: Timings{
			Send:    0,
			Wait:    elapsedMS,
			Receive: 0,
		},
	}
}

func NewRequest(req *http.Request, body []byte, opts ...Option) Request {
	config := applyOptions(opts)

	cookies := []Cookie{}
	if config.secureHeaders {
		cookies = NewCookies(req.Cookies())
	}

	return Request{
		Method:      req.Method,
		URL:         requestURL(req),
		HTTPVersion: req.Proto,
		Headers:     NewHeaders(req.Header, opts...),
		QueryString: NewQueryString(urlt.URL_Query(req.URL)),
		Cookies:     cookies,
		HeadersSize: -1,
		BodySize:    int64(len(body)),
		PostData:    NewPostData(body, req.Header.Get("Content-Type")),
	}
}

func NewResponse(resp *http.Response, body []byte, opts ...Option) Response {
	config := applyOptions(opts)

	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = defaultMimeType
	}

	cookies := []Cookie{}
	if config.secureHeaders {
		cookies = NewCookies(resp.Cookies())
	}

	return Response{
		Status:      resp.StatusCode,
		StatusText:  http.StatusText(resp.StatusCode),
		HTTPVersion: resp.Proto,
		Headers:     NewHeaders(resp.Header, opts...),
		Cookies:     cookies,
		Content:     NewContent(body, resp.Header.Get("Content-Encoding"), mimeType),
		RedirectURL: resp.Header.Get("Location"),
		HeadersSize: -1,
		BodySize:    int64(len(body)),
	}
}

func NewHeaders(header http.Header, opts ...Option) []NameValue {
	config := applyOptions(opts)
	result := make([]NameValue, 0, len(header))

	for name, values := range header {
		if !config.secureHeaders && IsSecureHeader(name) {
			continue
		}

		for _, value := range values {
			result = append(result, NameValue{Name: name, Value: value})
		}
	}

	return result
}

func NewQueryString(query url.Values) []NameValue {
	result := make([]NameValue, 0, len(query))

	for name, values := range query {
		for _, value := range values {
			result = append(result, NameValue{Name: name, Value: value})
		}
	}

	return result
}

func NewCookies(cookies []*http.Cookie) []Cookie {
	result := make([]Cookie, 0, len(cookies))

	for _, cookie := range cookies {
		result = append(result, Cookie{Name: cookie.Name, Value: cookie.Value})
	}

	return result
}

func NewPostData(body []byte, mimeType string) *PostData {
	if len(body) == 0 {
		return nil
	}

	postData := &PostData{MimeType: mimeType}
	if utf8.Valid(body) {
		postData.Text = string(body)
	}

	return postData
}

func requestURL(req *http.Request) string {
	if req.RequestURI == "" {
		return urlt.URL_String(req.URL)
	}

	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}

	return scheme + "://" + req.Host + req.RequestURI
}

func applyOptions(opts []Option) options {
	var config options

	for _, opt := range opts {
		opt(&config)
	}

	return config
}
