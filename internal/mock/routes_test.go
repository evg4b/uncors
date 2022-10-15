package mock_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/mock"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestMakeMockedRoutes(t *testing.T) {
	t.Run("Request method handling", func(t *testing.T) {
		t.Run("where mock method is not set allow method", func(t *testing.T) {
			router := mux.NewRouter()
			expectedBody := `{"name": "Jon Smite"}`
			mock.MakeMockedRoutes(router, []mock.Mock{{
				Path: "/api",
				Response: mock.Response{
					Code:       http.StatusOK,
					RawContent: expectedBody,
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

					response := recorder.Result()
					defer testutils.CheckNoError(t, response.Body.Close())

					body, err := io.ReadAll(response.Body)
					testutils.CheckNoError(t, err)

					assert.Equal(t, http.StatusOK, response.StatusCode)
					assert.Equal(t, expectedBody, string(body))
				})
			}
		})

		t.Run("where method is set", func(t *testing.T) {

		})
	})
	router := mux.NewRouter()
	mock.MakeMockedRoutes(router, []mock.Mock{
		{
			Path: "/api/user",
			Response: mock.Response{
				Code:       http.StatusOK,
				RawContent: `{"name": "Jon Smite"}`,
			},
		},
		{
			Path: "/api/bad_user",
			Response: mock.Response{
				Code:       http.StatusBadRequest,
				RawContent: `{"error": "incorrect data"}`,
			},
		},
	})

	tests := []struct {
		name       string
		method     string
		url        string
		headers    map[string]string
		expected   string
		statusCode int
	}{
		{
			name:       "http GET with status code 200",
			method:     http.MethodGet,
			url:        "http://localhost/api/user",
			expected:   `{"name": "Jon Smite"}`,
			statusCode: http.StatusOK,
		},
		{
			name:       "http GET with status code 400",
			method:     http.MethodGet,
			url:        "http://localhost/api/bad_user",
			expected:   `{"error": "incorrect data"}`,
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "http POST with status code 200",
			method:     http.MethodPost,
			url:        "http://localhost/api/user",
			expected:   `{"name": "Jon Smite"}`,
			statusCode: http.StatusOK,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(testCase.method, testCase.url, nil)
			for key, value := range testCase.headers {
				request.Header.Add(key, value)
			}
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			resp := recorder.Result()
			defer testutils.CheckNoError(t, resp.Body.Close())

			body, err := io.ReadAll(resp.Body)
			testutils.CheckNoError(t, err)

			assert.Equal(t, testCase.statusCode, resp.StatusCode)
			assert.Equal(t, testCase.expected, string(body))
		})
	}
}
