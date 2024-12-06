package mock_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/mock"
	"github.com/evg4b/uncors/pkg/fakedata"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func generateHandler(t *testing.T) (*mock.Handler, *mocks.GeneratorMock) {
	loggerMock := mocks.NewLoggerMock(t).
		PrintMock.
		Optional().
		Return()

	generatorMock := mocks.NewGeneratorMock(t)

	return mock.NewMockHandler(
		mock.WithLogger(loggerMock),
		mock.WithGenerator(generatorMock),
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
	), generatorMock
}

func TestFakeResponse(t *testing.T) {
	expectedString := "{\"hello\":\"world\",\"world\":\"hello\"}\n"
	responceObject := map[string]string{
		"hello": "world",
		"world": "hello",
	}

	t.Run("seed from query", func(t *testing.T) {
		handler, generatorMock := generateHandler(t)
		req := httptest.NewRequest(http.MethodGet, "/hello?$__uncors__seed=1", nil)

		generatorMock.GenerateMock.
			ExpectSeedParam2(1).
			Return(responceObject, nil)

		responseRecorder := httptest.NewRecorder()
		handler.ServeHTTP(contracts.WrapResponseWriter(responseRecorder), req)

		assert.Equal(t, http.StatusOK, responseRecorder.Code)
		assert.Equal(t, expectedString, testutils.ReadBody(t, responseRecorder))
	})

	t.Run("seed from header", func(t *testing.T) {
		handler, generatorMock := generateHandler(t)
		req := httptest.NewRequest(http.MethodGet, "/hello", nil)
		req.Header.Set("$__uncors__seed", "2")

		generatorMock.GenerateMock.
			ExpectSeedParam2(2).
			Return(responceObject, nil)

		responseRecorder := httptest.NewRecorder()
		handler.ServeHTTP(contracts.WrapResponseWriter(responseRecorder), req)

		assert.Equal(t, http.StatusOK, responseRecorder.Code)
		assert.Equal(t, expectedString, testutils.ReadBody(t, responseRecorder))
	})

	t.Run("invalid seed", func(t *testing.T) {
		handler, _ := generateHandler(t)
		req := httptest.NewRequest(http.MethodGet, "/hello?$__uncors__seed=invalid", nil)

		responseRecorder := httptest.NewRecorder()
		handler.ServeHTTP(contracts.WrapResponseWriter(responseRecorder), req)

		assert.Equal(t, http.StatusInternalServerError, responseRecorder.Code)
		assert.Contains(t, testutils.ReadBody(t, responseRecorder), "invalid $__uncors__seed parameter")
	})

	t.Run("generation failed", func(t *testing.T) {
		handler, generatorMock := generateHandler(t)
		req := httptest.NewRequest(http.MethodGet, "/hello", nil)
		testErr := errors.ErrUnsupported
		generatorMock.GenerateMock.Return(responceObject, testErr)

		responseRecorder := httptest.NewRecorder()
		handler.ServeHTTP(contracts.WrapResponseWriter(responseRecorder), req)

		assert.Equal(t, http.StatusInternalServerError, responseRecorder.Code)
		assert.Contains(t, testutils.ReadBody(t, responseRecorder), "Occurred error: unsupported operation")
		assert.Len(t, generatorMock.GenerateMock.Calls(), 1)
	})
}
