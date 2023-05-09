package static_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/handler/static"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

const (
	indexJS   = "/assets/index.js"
	demoJS    = "/assets/demo.js"
	indexHTML = "/index.html"
)
const (
	indexJSContent   = "console.log('index.js')"
	demoJSContent    = "console.log('demo.js')"
	indexHTMLContent = "<html!></html>"
)

func TestMiddleware(t *testing.T) {
	fs := testutils.FsFromMap(t, map[string]string{
		indexJS:   indexJSContent,
		demoJS:    demoJSContent,
		indexHTML: indexHTMLContent,
	})

	staticFileTests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "should send pain file",
			path:     indexHTML,
			expected: indexHTMLContent,
		},
		{
			name:     "should send file from folder",
			path:     indexJS,
			expected: indexJSContent,
		},
		{
			name:     "should send file ignore query params",
			path:     indexHTML + "?testParam=test",
			expected: indexHTMLContent,
		},
		{
			name:     "should send file from folder ignore query params",
			path:     demoJS + "?testParam=test",
			expected: demoJSContent,
		},
		{
			name:     "should send pain file without root slash",
			path:     strings.TrimPrefix(indexHTML, "/"),
			expected: indexHTMLContent,
		},
		{
			name:     "should send demo.js file from folder without root slash",
			path:     strings.TrimPrefix(demoJS, "/"),
			expected: demoJSContent,
		},
	}

	notExistingFilesTests := []struct {
		name string
		path string
	}{
		{
			name: "where file not exists",
			path: "/options.html",
		},
		{
			name: "where request directory",
			path: "/assets/",
		},
		{
			name: "where request directory without trailing slash",
			path: "/assets",
		},
		{
			name: "where request not exists directory",
			path: "/options/",
		},
	}

	t.Run("index file is not configured", func(t *testing.T) {
		const testHTTPStatusCode = 999
		const testHTTPBody = "this is tests response"
		middleware := static.NewStaticMiddleware(
			static.WithFileSystem(fs),
			static.WithLogger(mocks.NewLoggerMock(t)),
			static.WithNext(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(testHTTPStatusCode)
				fmt.Fprint(writer, testHTTPBody)
			})),
		)

		t.Run("return static content", func(t *testing.T) {
			for _, testCase := range staticFileTests {
				t.Run(testCase.name, func(t *testing.T) {
					recorder := httptest.NewRecorder()
					requestURI, err := url.Parse(testCase.path)
					testutils.CheckNoError(t, err)

					middleware.ServeHTTP(recorder, &http.Request{
						Method: http.MethodGet,
						URL:    requestURI,
					})

					assert.Equal(t, recorder.Code, http.StatusOK)
					assert.Equal(t, testCase.expected, testutils.ReadBody(t, recorder))
				})
			}
		})

		t.Run("forward request to next handler", func(t *testing.T) {
			for _, testCase := range notExistingFilesTests {
				t.Run(testCase.name, func(t *testing.T) {
					recorder := httptest.NewRecorder()
					requestURI, err := url.Parse(testCase.path)
					testutils.CheckNoError(t, err)

					middleware.ServeHTTP(recorder, &http.Request{
						Method: http.MethodGet,
						URL:    requestURI,
					})

					assert.Equal(t, testHTTPStatusCode, recorder.Code)
					assert.Equal(t, testHTTPBody, testutils.ReadBody(t, recorder))
				})
			}
		})
	})

	t.Run("index file is configured", func(t *testing.T) {
		middleware := static.NewStaticMiddleware(
			static.WithFileSystem(fs),
			static.WithLogger(mocks.NewLoggerMock(t)),
			static.WithIndex(indexHTML),
			static.WithNext(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				panic("should not be called")
			})),
		)

		t.Run("send index file", func(t *testing.T) {
			for _, testCase := range staticFileTests {
				t.Run(testCase.name, func(t *testing.T) {
					recorder := httptest.NewRecorder()
					requestURI, err := url.Parse(testCase.path)
					testutils.CheckNoError(t, err)

					middleware.ServeHTTP(recorder, &http.Request{
						Method: http.MethodGet,
						URL:    requestURI,
					})

					assert.Equal(t, recorder.Code, http.StatusOK)
					assert.Equal(t, testCase.expected, testutils.ReadBody(t, recorder))
				})
			}
		})

		t.Run("forward request to next handler", func(t *testing.T) {
			for _, testCase := range notExistingFilesTests {
				t.Run(testCase.name, func(t *testing.T) {
					recorder := httptest.NewRecorder()
					requestURI, err := url.Parse(testCase.path)
					testutils.CheckNoError(t, err)

					middleware.ServeHTTP(recorder, &http.Request{
						Method: http.MethodGet,
						URL:    requestURI,
					})

					assert.Equal(t, http.StatusOK, recorder.Code)
					assert.Equal(t, indexHTMLContent, testutils.ReadBody(t, recorder))
				})
			}
		})

		t.Run("index file doesn't exists", func(t *testing.T) {
			middleware := static.NewStaticMiddleware(
				static.WithFileSystem(fs),
				static.WithLogger(mocks.NewLoggerMock(t)),
				static.WithIndex("/not-exists.html"),
				static.WithNext(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
					panic("should not be called")
				})),
			)

			recorder := httptest.NewRecorder()
			requestURI, err := url.Parse("/options/")
			testutils.CheckNoError(t, err)

			middleware.ServeHTTP(recorder, &http.Request{
				Method: http.MethodGet,
				URL:    requestURI,
			})

			assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		})
	})
}