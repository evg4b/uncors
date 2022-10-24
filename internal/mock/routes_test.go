//nolint:maintidx
package mock_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/mock"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

var mock1Body = `{"mock": "mock number 1"}`
var mock2Body = `{"mock": "mock number 2"}`
var mock3Body = `{"mock": "mock number 3"}`
var mock4Body = `{"mock": "mock number 4"}`

func TestMakeMockedRoutes(t *testing.T) {
	logger := mocks.NewNoopLogger(t)

	t.Run("request method handling", func(t *testing.T) {
		t.Run("where mock method is not set allow method", func(t *testing.T) {
			router := mux.NewRouter()
			mock.MakeMockedRoutes(router, logger, []mock.Mock{{
				Path: "/api",
				Response: mock.Response{
					Code:       http.StatusOK,
					RawContent: mock1Body,
				},
			}})

			methods := []string{
				http.MethodGet,
				http.MethodHead,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
				http.MethodOptions,
				http.MethodTrace,
			}

			for _, method := range methods {
				t.Run(method, func(t *testing.T) {
					request := httptest.NewRequest(method, "http://localhost/api", nil)
					recorder := httptest.NewRecorder()

					router.ServeHTTP(recorder, request)

					body := testutils.ReadBody(t, recorder)
					assert.Equal(t, http.StatusOK, recorder.Code)
					assert.Equal(t, mock1Body, body)
				})
			}
		})

		t.Run("where method is set", func(t *testing.T) {
			router := mux.NewRouter()
			mock.MakeMockedRoutes(router, logger, []mock.Mock{{
				Path:   "/api",
				Method: http.MethodPut,
				Response: mock.Response{
					Code:       http.StatusOK,
					RawContent: mock1Body,
				},
			}})

			t.Run("method is not matched", func(t *testing.T) {
				methods := []string{
					http.MethodGet,
					http.MethodHead,
					http.MethodPost,
					http.MethodPatch,
					http.MethodDelete,
					http.MethodOptions,
					http.MethodTrace,
				}

				for _, method := range methods {
					t.Run(method, func(t *testing.T) {
						request := httptest.NewRequest(method, "http://localhost/api", nil)
						recorder := httptest.NewRecorder()

						router.ServeHTTP(recorder, request)

						assert.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
					})
				}
			})

			t.Run("method is matched", func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPut, "http://localhost/api", nil)
				recorder := httptest.NewRecorder()

				router.ServeHTTP(recorder, request)

				body := testutils.ReadBody(t, recorder)
				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, mock1Body, body)
			})
		})
	})

	t.Run("path handling", func(t *testing.T) {
		router := mux.NewRouter()

		mock.MakeMockedRoutes(router, logger, []mock.Mock{
			{
				Path: "/api/user",
				Response: mock.Response{
					Code:       http.StatusOK,
					RawContent: mock1Body,
				},
			},
			{
				Path: "/api/user/{id:[0-9]+}",
				Response: mock.Response{
					Code:       http.StatusAccepted,
					RawContent: mock2Body,
				},
			},
			{
				Path: "/api/{single-path/demo",
				Response: mock.Response{
					Code:       http.StatusBadRequest,
					RawContent: mock3Body,
				},
			},
			{
				Path: `/api/v2/{multiple-path:[a-z-\/]+}/demo`,
				Response: mock.Response{
					Code:       http.StatusCreated,
					RawContent: mock4Body,
				},
			},
		})

		tests := []struct {
			name       string
			url        string
			expected   string
			statusCode int
		}{
			{
				name:       "direct path",
				url:        "https://localhost/api/user",
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name:       "direct path with ending slash",
				url:        "https://localhost/api/user/",
				expected:   "404 page not found\n",
				statusCode: http.StatusNotFound,
			},
			{
				name:       "direct path with parameter",
				url:        "https://localhost/api/user/23",
				expected:   mock2Body,
				statusCode: http.StatusAccepted,
			},
			{
				name:       "direct path with incorrect parameter",
				url:        "https://localhost/api/user/unknow",
				expected:   "404 page not found\n",
				statusCode: http.StatusNotFound,
			},
			{
				name:       "path with subpath to single matching param",
				url:        "https://localhost/api/some-path/with-some-subpath/demo",
				expected:   "404 page not found\n",
				statusCode: http.StatusNotFound,
			},
			{
				name:       "path with subpath to multiple matching param",
				url:        "https://localhost/api/v2/some-path/with-some-subpath/demo",
				expected:   mock4Body,
				statusCode: http.StatusCreated,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodGet, testCase.url, nil)
				recorder := httptest.NewRecorder()

				router.ServeHTTP(recorder, request)

				body := testutils.ReadBody(t, recorder)
				assert.Equal(t, testCase.statusCode, recorder.Code)
				assert.Equal(t, testCase.expected, body)
			})
		}
	})

	t.Run("query handling", func(t *testing.T) {
		router := mux.NewRouter()
		mock.MakeMockedRoutes(router, logger, []mock.Mock{
			{
				Path: "/api/user",
				Response: mock.Response{
					Code:       http.StatusOK,
					RawContent: mock1Body,
				},
			},
			{
				Path: "/api/user",
				Queries: map[string]string{
					"id": "17",
				},
				Response: mock.Response{
					Code:       http.StatusCreated,
					RawContent: mock2Body,
				},
			},
			{
				Path: "/api/user",
				Queries: map[string]string{
					"id":    "99",
					"token": "fe145b54563d9be1b2a476f56b0a412b",
				},
				Response: mock.Response{
					Code:       http.StatusAccepted,
					RawContent: mock3Body,
				},
			},
		})

		tests := []struct {
			name       string
			url        string
			expected   string
			statusCode int
		}{
			{
				name:       "queries is not set",
				url:        "http://localhost/api/user",
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name:       "passed unsetted parameter",
				url:        "http://localhost/api/user?id=16",
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name:       "passed parameter",
				url:        "http://localhost/api/user?id=17",
				expected:   mock2Body,
				statusCode: http.StatusCreated,
			},
			{
				name:       "passed one of multiple parameters",
				url:        "http://localhost/api/user?id=99",
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name:       "passed all of multiple parameters",
				url:        "http://localhost/api/user?id=99&token=fe145b54563d9be1b2a476f56b0a412b",
				expected:   mock3Body,
				statusCode: http.StatusAccepted,
			},
			{
				name:       "passed extra parameters",
				url:        "http://localhost/api/user?id=99&token=fe145b54563d9be1b2a476f56b0a412b&demo=true",
				expected:   mock3Body,
				statusCode: http.StatusAccepted,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodGet, testCase.url, nil)
				recorder := httptest.NewRecorder()

				router.ServeHTTP(recorder, request)

				body := testutils.ReadBody(t, recorder)
				assert.Equal(t, testCase.statusCode, recorder.Code)
				assert.Equal(t, testCase.expected, body)
			})
		}
	})

	t.Run("header handling", func(t *testing.T) {
		router := mux.NewRouter()
		mock.MakeMockedRoutes(router, logger, []mock.Mock{
			{
				Path: "/api/user",
				Response: mock.Response{
					Code:       http.StatusOK,
					RawContent: mock1Body,
				},
			},
			{
				Path: "/api/user",
				Headers: map[string]string{
					"Token": "de4e27987d054577b0edc0e828851724",
				},
				Response: mock.Response{
					Code:       http.StatusCreated,
					RawContent: mock2Body,
				},
			},
			{
				Path: "/api/user",
				Headers: map[string]string{
					"User-Id": "99",
					"Token":   "fe145b54563d9be1b2a476f56b0a412b",
				},
				Response: mock.Response{
					Code:       http.StatusAccepted,
					RawContent: mock3Body,
				},
			},
		})

		tests := []struct {
			name       string
			url        string
			headers    map[string]string
			expected   string
			statusCode int
		}{
			{
				name:       "headers is not set",
				url:        "https://localhost/api/user",
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name: "passed unsetted headers",
				url:  "https://localhost/api/user",
				headers: map[string]string{
					"Token": "55cc413b96026e833835a2c9a3f39c21",
				},
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name: "passed defined header",
				url:  "https://localhost/api/user",
				headers: map[string]string{
					"Token": "de4e27987d054577b0edc0e828851724",
				},
				expected:   mock2Body,
				statusCode: http.StatusCreated,
			},
			{
				name: "passed one of multiple headers",
				url:  "https://localhost/api/user",
				headers: map[string]string{
					"User-Id": "99",
				},
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name: "passed all of multiple headers",
				url:  "https://localhost/api/user",
				headers: map[string]string{
					"User-Id": "99",
					"Token":   "fe145b54563d9be1b2a476f56b0a412b",
				},
				expected:   mock3Body,
				statusCode: http.StatusAccepted,
			},
			{
				name: "passed extra headers",
				url:  "https://localhost/api/user",
				headers: map[string]string{
					"User-Id":         "99",
					"Token":           "fe145b54563d9be1b2a476f56b0a412b",
					"Accept-Encoding": "deflate",
				},
				expected:   mock3Body,
				statusCode: http.StatusAccepted,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPost, testCase.url, nil)
				for key, value := range testCase.headers {
					request.Header.Add(key, value)
				}
				recorder := httptest.NewRecorder()

				router.ServeHTTP(recorder, request)

				body := testutils.ReadBody(t, recorder)
				assert.Equal(t, testCase.statusCode, recorder.Code)
				assert.Equal(t, testCase.expected, body)
			})
		}
	})
}
