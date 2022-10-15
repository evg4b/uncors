package mock_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/mock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestMakeMockedRoutes(t *testing.T) {
	router := mux.NewRouter()
	mock.MakeMockedRoutes(router, []mock.Mock{
		{
			Path: "/api/user",
			Response: mock.Response{
				Code:       http.StatusOK,
				RawContent: `{"name": "Jon Smite"}`,
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
			name:       "",
			method:     http.MethodGet,
			url:        "/api/user",
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
			defer resp.Body.Close()

			assert.Equal(t, testCase.statusCode, resp.StatusCode)
		})
	}
}
