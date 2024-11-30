package mock_test

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/mock"
	"github.com/evg4b/uncors/pkg/fakedata"
	"github.com/evg4b/uncors/testing/mocks"
)

func TestFakeResponse(t *testing.T) {
	loggerMock := mocks.NewLoggerMock(t).
		PrintMock.Return()

	handler := mock.NewMockHandler(
		mock.WithLogger(loggerMock),
		mock.WithResponse(config.Response{
			Code: http.StatusOK,
			Fake: &fakedata.Node{
				Type: "object",
				Properties: map[string]fakedata.Node{
					"hello": {
						Type: "string",
						Options: map[string]interface{}{
							"wordcount": 3,
						},
					},
					"world": {
						Type: "string",
						Options: map[string]interface{}{
							"wordcount": 3,
						},
					},
				},
			},
		}),
	)

	t.Run("seed from query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/hello?$__uncors__seed=123", nil)

		responseRecorder := httptest.NewRecorder()
		handler.ServeHTTP(contracts.WrapResponseWriter(responseRecorder), req)

		assert.Equal(t, responseRecorder.Code, http.StatusOK)
		actual := responseRecorder.Body.String()
		assert.Equal(t, "{\"hello\":\"At esse ea.\",\"world\":\"Sint ut culpa.\"}\n", actual)
	})

	t.Run("seed from header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/hello", nil)
		req.Header.Set("$__uncors__seed", "123")

		responseRecorder := httptest.NewRecorder()
		handler.ServeHTTP(contracts.WrapResponseWriter(responseRecorder), req)

		assert.Equal(t, responseRecorder.Code, http.StatusOK)
		actual := responseRecorder.Body.String()
		assert.Equal(t, "{\"hello\":\"At esse ea.\",\"world\":\"Sint ut culpa.\"}\n", actual)
	})

	t.Run("invalid seed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/hello?$__uncors__seed=invalid", nil)

		responseRecorder := httptest.NewRecorder()
		handler.ServeHTTP(contracts.WrapResponseWriter(responseRecorder), req)

		assert.Equal(t, http.StatusInternalServerError, responseRecorder.Code)
		actual := responseRecorder.Body.String()
		assert.Contains(t, actual, "invalid $__uncors__seed parameter")
	})
}
