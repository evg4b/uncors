package infra_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCastToHTTPHandler(t *testing.T) {
	const expectedBody = `{ "OK": true }`

	handlerStub := infra.HandlerFunc(func(w contracts.ResponseWriter, _ *contracts.Request) error {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, expectedBody)

		return nil
	})

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/data", nil)
	handler := infra.CastToHTTPHandler(handlerStub)

	t.Run("cast correctly", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		responseWriter := server.NewResponseRecorder(recorder)

		assert.NotPanics(t, func() {
			handler.ServeHTTP(responseWriter, request)
			assert.Equal(t, expectedBody, testutils.ReadBody(t, recorder))
		})
	})

	t.Run("panic when request is not wrapped", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		assert.PanicsWithValue(t, infra.ErrResponseNotCasted, func() {
			handler.ServeHTTP(recorder, request)
		})
	})
}

func TestHandlerFunc(t *testing.T) {
	const expectedBody = `{ "OK": true }`

	uncorsHandler := infra.HandlerFunc(func(w contracts.ResponseWriter, _ *contracts.Request) error {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, expectedBody)

		return nil
	})

	recorder := httptest.NewRecorder()
	responseWriter := server.NewResponseRecorder(recorder)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/data", nil)

	err := uncorsHandler.ServeHTTP(responseWriter, request)

	require.NoError(t, err)
	assert.Equal(t, expectedBody, testutils.ReadBody(t, recorder))
}
