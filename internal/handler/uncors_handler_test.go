package handler_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/helpers"

	"github.com/evg4b/uncors/internal/configuration"
	"github.com/evg4b/uncors/internal/handler"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestUncorsRequestHandler(t *testing.T) {
	fs := testutils.FsFromMap(t, map[string]string{
		"/images/background.png": "background.png",
		"/images/svg/icons.svg":  "icons.svg",
		"/assets/js/index.js":    "index.js",
		"/assets/css/styles.css": "styles.css",
		"/assets/index.html":     "index.html",
		"/mock.json":             "mock.json",
	})

	mappings := []configuration.URLMapping{
		{
			From: "http://localhost",
			To:   "https://localhost",
			Statics: []configuration.StaticDirMapping{
				{Dir: "/assets", Path: "/cc/", Index: "index.html"},
				{Dir: "/assets", Path: "/pnp/", Index: "index.php"},
				{Dir: "/images", Path: "/img/"},
			},
		},
	}

	mockDefs := []configuration.Mock{
		{
			Path: "/api/mocks/1",
			Response: configuration.Response{
				Code:       http.StatusOK,
				RawContent: "mock-1",
			},
		},
		{
			Path: "/api/mocks/2",
			Response: configuration.Response{
				Code: http.StatusOK,
				File: "/mock.json",
			},
		},
		{
			Path: "/api/mocks/3",
			Response: configuration.Response{
				Code:       http.StatusMultiStatus,
				RawContent: "mock-3",
			},
		},
		{
			Path: "/api/mocks/4",
			Response: configuration.Response{
				Code: http.StatusOK,
				File: "/unknown.json",
			},
		},
	}

	factory, err := urlreplacer.NewURLReplacerFactory(mappings)
	testutils.CheckNoError(t, err)

	httpResponseMapping := map[string]string{
		"/img/original.png": "original.png",
	}

	httpMock := mocks.NewHTTPClientMock(t).DoMock.Set(func(request *http.Request) (*http.Response, error) {
		if response, ok := httpResponseMapping[request.URL.Path]; ok {
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(response)),
				Status:     http.StatusText(http.StatusOK),
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Request:    request,
			}, nil
		}

		panic(fmt.Sprintf("incorrect request: %s", request.URL.Path))
	})

	hand := handler.NewUncorsRequestHandler(
		handler.WithLogger(mocks.NewLoggerMock(t)),
		handler.WithMocks(mockDefs),
		handler.WithFileSystem(fs),
		handler.WithURLReplacerFactory(factory),
		handler.WithHTTPClient(httpMock),
		handler.WithMappings(mappings),
	)

	t.Run("statics directory", func(t *testing.T) {
		t.Run("with index file", func(t *testing.T) {
			t.Run("should return static file", func(t *testing.T) {
				tests := []struct {
					name     string
					url      string
					expected string
				}{
					{
						name:     "index.html",
						url:      "http://localhost/cc/index.html",
						expected: "index.html",
					},
					{
						name:     "index.js",
						url:      "http://localhost/cc/js/index.js",
						expected: "index.js",
					},
					{
						name:     "styles.css",
						url:      "http://localhost/cc/css/styles.css",
						expected: "styles.css",
					},
				}
				for _, testCase := range tests {
					t.Run(testCase.name, func(t *testing.T) {
						recorder := httptest.NewRecorder()
						request := httptest.NewRequest(http.MethodGet, testCase.url, nil)
						helpers.NormaliseRequest(request)

						hand.ServeHTTP(recorder, request)

						assert.Equal(t, 200, recorder.Code)
						assert.Equal(t, testCase.expected, testutils.ReadBody(t, recorder))
					})
				}
			})

			t.Run("should return index file by default", func(t *testing.T) {
				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "http://localhost/cc/unknown.html", nil)
				helpers.NormaliseRequest(request)

				hand.ServeHTTP(recorder, request)

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, "index.html", testutils.ReadBody(t, recorder))
			})

			t.Run("should return error code when index file doesn't exists", func(t *testing.T) {
				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "http://localhost/pnp/unknown.html", nil)
				helpers.NormaliseRequest(request)

				hand.ServeHTTP(recorder, request)

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assert.Contains(t, testutils.ReadBody(t, recorder), "Internal Server Error")
			})
		})

		t.Run("without index file", func(t *testing.T) {
			t.Run("should return static file", func(t *testing.T) {
				tests := []struct {
					name     string
					url      string
					expected string
				}{
					{
						name:     "background.png",
						url:      "http://localhost/img/background.png",
						expected: "background.png",
					},
					{
						name:     "icons.svg",
						url:      "http://localhost/img/svg/icons.svg",
						expected: "icons.svg",
					},
				}
				for _, testCase := range tests {
					t.Run(testCase.name, func(t *testing.T) {
						recorder := httptest.NewRecorder()
						request := httptest.NewRequest(http.MethodGet, testCase.url, nil)
						helpers.NormaliseRequest(request)

						hand.ServeHTTP(recorder, request)

						assert.Equal(t, http.StatusOK, recorder.Code)
						assert.Equal(t, testCase.expected, testutils.ReadBody(t, recorder))
					})
				}
			})

			t.Run("should return original file", func(t *testing.T) {
				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "http://localhost/img/original.png", nil)
				helpers.NormaliseRequest(request)

				hand.ServeHTTP(recorder, request)

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, "original.png", testutils.ReadBody(t, recorder))
			})
		})
	})

	t.Run("mocks", func(t *testing.T) {
		t.Run("should return mock with", func(t *testing.T) {
			tests := []struct {
				name         string
				url          string
				expected     string
				expectedCode int
			}{
				{
					name:         "raw content mock",
					url:          "http://localhost/api/mocks/1",
					expected:     "mock-1",
					expectedCode: http.StatusOK,
				},
				{
					name:         "file content mock",
					url:          "http://localhost/api/mocks/2",
					expected:     "mock.json",
					expectedCode: http.StatusOK,
				},
				{
					name:         "MultiStatus mock",
					url:          "http://localhost/api/mocks/3",
					expected:     "mock-3",
					expectedCode: http.StatusMultiStatus,
				},
			}
			for _, testCase := range tests {
				t.Run(testCase.name, func(t *testing.T) {
					recorder := httptest.NewRecorder()
					request := httptest.NewRequest(http.MethodGet, testCase.url, nil)
					helpers.NormaliseRequest(request)

					hand.ServeHTTP(recorder, request)

					assert.Equal(t, testCase.expectedCode, recorder.Code)
					assert.Equal(t, testCase.expected, testutils.ReadBody(t, recorder))
				})
			}
		})

		t.Run("should return error code when mock file doesn't exists", func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "http://localhost/api/mocks/4", nil)
			helpers.NormaliseRequest(request)

			hand.ServeHTTP(recorder, request)

			assert.Equal(t, http.StatusInternalServerError, recorder.Code)
			assert.Contains(t, testutils.ReadBody(t, recorder), "Internal Server Error")
		})
	})
}
