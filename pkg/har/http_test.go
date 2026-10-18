package har_test

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/evg4b/uncors/pkg/har"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSecureHeader(t *testing.T) {
	for _, name := range []string{
		"Cookie", "set-cookie", "AUTHORIZATION", "WWW-Authenticate", "Proxy-Authorization", "proxy-authenticate",
	} {
		assert.True(t, har.IsSecureHeader(name), name)
	}

	for _, name := range []string{"Content-Type", "Accept", "X-Api-Key"} {
		assert.False(t, har.IsSecureHeader(name), name)
	}
}

func TestNewEntry(t *testing.T) {
	start := time.Date(2026, time.October, 16, 12, 0, 0, 0, time.UTC)
	request := har.Request{Method: http.MethodGet}
	response := har.Response{Status: http.StatusOK}

	entry := har.NewEntry(start, 1500*time.Microsecond, request, response)

	assert.Equal(t, har.Entry{
		StartedDateTime: start,
		Time:            1.5,
		Request:         request,
		Response:        response,
		Timings:         har.Timings{Wait: 1.5},
	}, entry)
}

func TestNewRequest(t *testing.T) {
	t.Run("records a server request", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/items?page=2", strings.NewReader(`{"a":1}`))
		req.Host = "example.com:8080"
		req.Header.Set("Content-Type", "application/json")

		request := har.NewRequest(req, []byte(`{"a":1}`))

		assert.Equal(t, har.Request{
			Method:      http.MethodPost,
			URL:         "http://example.com:8080/api/items?page=2",
			HTTPVersion: "HTTP/1.1",
			Headers:     []har.NameValue{{Name: "Content-Type", Value: "application/json"}},
			QueryString: []har.NameValue{{Name: "page", Value: "2"}},
			Cookies:     []har.Cookie{},
			HeadersSize: -1,
			BodySize:    7,
			PostData:    &har.PostData{MimeType: "application/json", Text: `{"a":1}`},
		}, request)
	})

	t.Run("uses https for a TLS connection", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.TLS = &tls.ConnectionState{}

		assert.Equal(t, "https://example.com/", har.NewRequest(req, nil).URL)
	})

	t.Run("records a client request", func(t *testing.T) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com/a?b=c", nil)
		require.NoError(t, err)

		assert.Equal(t, "https://example.com/a?b=c", har.NewRequest(req, nil).URL)
	})

	t.Run("leaves out secure headers and cookies by default", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer token")
		req.AddCookie(&http.Cookie{Name: "session", Value: "abc"}) // nolint: gosec

		request := har.NewRequest(req, nil)

		assert.Empty(t, request.Headers)
		assert.Equal(t, []har.Cookie{}, request.Cookies)
	})

	t.Run("captures secure headers and cookies on request", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: "abc"}) // nolint: gosec

		request := har.NewRequest(req, nil, har.WithSecureHeaders(true))

		assert.Equal(t, []har.NameValue{{Name: "Cookie", Value: "session=abc"}}, request.Headers)
		assert.Equal(t, []har.Cookie{{Name: "session", Value: "abc"}}, request.Cookies)
	})
}

func TestNewResponse(t *testing.T) {
	t.Run("records a response", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusFound,
			Proto:      "HTTP/2.0",
			Header: http.Header{
				"Content-Type": {"text/plain"},
				"Location":     {"/next"},
			},
		}

		response := har.NewResponse(resp, []byte("moved"))

		assert.Equal(t, http.StatusFound, response.Status)
		assert.Equal(t, "Found", response.StatusText)
		assert.Equal(t, "HTTP/2.0", response.HTTPVersion)
		assert.Equal(t, "/next", response.RedirectURL)
		assert.Equal(t, har.Content{Size: 5, MimeType: "text/plain", Text: "moved"}, response.Content)
		assert.Equal(t, int64(5), response.BodySize)
		assert.Equal(t, -1, response.HeadersSize)
		assert.Len(t, response.Headers, 2)
	})

	t.Run("defaults the MIME type", func(t *testing.T) {
		response := har.NewResponse(&http.Response{StatusCode: http.StatusOK, Header: http.Header{}}, nil)

		assert.Equal(t, "application/octet-stream", response.Content.MimeType)
	})

	t.Run("records the size of the body as sent", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Encoding": {"gzip"}},
		}
		body := gzipped(t, "hello")

		response := har.NewResponse(resp, body)

		assert.Equal(t, int64(len(body)), response.BodySize)
		assert.Equal(t, int64(5), response.Content.Size)
		assert.Equal(t, "hello", response.Content.Text)
	})

	t.Run("captures cookies only on request", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Set-Cookie": {"session=abc; Path=/; HttpOnly"}},
		}

		assert.Empty(t, har.NewResponse(resp, nil).Headers)
		assert.Equal(t, []har.Cookie{}, har.NewResponse(resp, nil).Cookies)

		captured := har.NewResponse(resp, nil, har.WithSecureHeaders(true))
		assert.Equal(t, []har.NameValue{{Name: "Set-Cookie", Value: "session=abc; Path=/; HttpOnly"}}, captured.Headers)
		assert.Equal(t, []har.Cookie{{Name: "session", Value: "abc", Path: "/", HTTPOnly: true}}, captured.Cookies)
	})
}

func TestNewHeaders(t *testing.T) {
	headers := har.NewHeaders(http.Header{
		"X-Trace": {"1"},
		"Accept":  {"text/html", "application/json"},
		"Cookie":  {"session=abc"},
	})

	assert.Equal(t, []har.NameValue{
		{Name: "Accept", Value: "text/html"},
		{Name: "Accept", Value: "application/json"},
		{Name: "X-Trace", Value: "1"},
	}, headers)
}

func TestNewQueryString(t *testing.T) {
	t.Run("keeps the order of the parameters", func(t *testing.T) {
		query := har.NewQueryString("z=1&a=2&z=3&b&q=hello+world%21")

		assert.Equal(t, []har.NameValue{
			{Name: "z", Value: "1"},
			{Name: "a", Value: "2"},
			{Name: "z", Value: "3"},
			{Name: "b", Value: ""},
			{Name: "q", Value: "hello world!"},
		}, query)
	})

	t.Run("keeps a malformed escape as it is", func(t *testing.T) {
		assert.Equal(t, []har.NameValue{{Name: "a", Value: "%zz"}}, har.NewQueryString("a=%zz"))
	})

	t.Run("without a query", func(t *testing.T) {
		assert.Equal(t, []har.NameValue{}, har.NewQueryString(""))
	})
}

func TestNewCookies(t *testing.T) {
	expires := time.Date(2026, time.October, 16, 12, 0, 0, 0, time.UTC)

	cookies := har.NewCookies([]*http.Cookie{
		{Name: "plain", Value: "1"}, // nolint: gosec
		{ // nolint: gosec
			Name:     "session",
			Value:    "abc",
			Path:     "/api",
			Domain:   "example.com",
			Expires:  expires,
			HttpOnly: true,
			Secure:   true,
		},
	})

	assert.Equal(t, []har.Cookie{
		{Name: "plain", Value: "1"},
		{
			Name:     "session",
			Value:    "abc",
			Path:     "/api",
			Domain:   "example.com",
			Expires:  "2026-10-16T12:00:00.000Z",
			HTTPOnly: true,
			Secure:   true,
		},
	}, cookies)
}

func TestNewPostData(t *testing.T) {
	t.Run("without a body", func(t *testing.T) {
		assert.Nil(t, har.NewPostData(nil, "text/plain"))
	})

	t.Run("with a text body", func(t *testing.T) {
		assert.Equal(t, &har.PostData{MimeType: "text/plain", Text: "hi"}, har.NewPostData([]byte("hi"), "text/plain"))
	})

	t.Run("with a URL-encoded form", func(t *testing.T) {
		postData := har.NewPostData([]byte("name=John+Doe&tag=a&tag=b"), "application/x-www-form-urlencoded; charset=UTF-8")

		assert.Equal(t, &har.PostData{
			MimeType: "application/x-www-form-urlencoded; charset=UTF-8",
			Params: []har.Param{
				{Name: "name", Value: "John Doe"},
				{Name: "tag", Value: "a"},
				{Name: "tag", Value: "b"},
			},
		}, postData)
	})

	t.Run("with a binary body", func(t *testing.T) {
		assert.Equal(t, &har.PostData{MimeType: "image/png"}, har.NewPostData([]byte{0xff, 0xfe}, "image/png"))
	})
}
