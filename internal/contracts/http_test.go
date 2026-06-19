package contracts_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestCastToHTTPHandler(t *testing.T) {
	const expectedBody = `{ "OK": true }`

	handlerStub := contracts.HandlerFunc(func(w contracts.ResponseWriter, _ *contracts.Request) error {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, expectedBody)

		return nil
	})

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/data", nil)
	handler := contracts.CastToHTTPHandler(handlerStub)

	t.Run("cast correctly", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		responseWriter := contracts.NewResponseRecorder(recorder)

		assert.NotPanics(t, func() {
			handler.ServeHTTP(responseWriter, request) //nolint:errcheck
			assert.Equal(t, expectedBody, testutils.ReadBody(t, recorder))
		})
	})

	t.Run("panic when request is not wrapped", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		assert.PanicsWithValue(t, contracts.ErrResponseNotCasted, func() {
			handler.ServeHTTP(recorder, request) //nolint:errcheck
		})
	})
}

func TestHandlerFunc(t *testing.T) {
	const expectedBody = `{ "OK": true }`

	uncorsHandler := contracts.HandlerFunc(func(w contracts.ResponseWriter, _ *contracts.Request) error {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, expectedBody)

		return nil
	})

	recorder := httptest.NewRecorder()
	responseWriter := contracts.NewResponseRecorder(recorder)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/data", nil)

	uncorsHandler.ServeHTTP(responseWriter, request) //nolint:errcheck

	assert.Equal(t, expectedBody, testutils.ReadBody(t, recorder))
}
