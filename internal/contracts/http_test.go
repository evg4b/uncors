package contracts_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/sfmt"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestCastToHTTPHandler(t *testing.T) {
	const expectedBody = `{ "OK": true }`
	uncorsHandler := contracts.HandlerFunc(func(w contracts.ResponseWriter, r *contracts.Request) {
		w.WriteHeader(http.StatusOK)
		sfmt.Fprint(w, expectedBody)
	})

	request := httptest.NewRequest(http.MethodGet, "/data", nil)
	httpHandler := contracts.CastToHTTPHandler(uncorsHandler)

	t.Run("cast correctly", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		responseWriter := contracts.WrapResponseWriter(recorder)

		assert.NotPanics(t, func() {
			httpHandler.ServeHTTP(responseWriter, request)
			assert.Equal(t, expectedBody, testutils.ReadBody(t, recorder))
		})
	})

	t.Run("panic when request is not wrapped", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		assert.PanicsWithValue(t, contracts.ErrResponseNotCasted, func() {
			httpHandler.ServeHTTP(recorder, request)
		})
	})
}

func TestHandlerFunc(t *testing.T) {
	const expectedBody = `{ "OK": true }`
	uncorsHandler := contracts.HandlerFunc(func(w contracts.ResponseWriter, r *contracts.Request) {
		w.WriteHeader(http.StatusOK)
		sfmt.Fprint(w, expectedBody)
	})

	recorder := httptest.NewRecorder()
	responseWriter := contracts.WrapResponseWriter(recorder)
	request := httptest.NewRequest(http.MethodGet, "/data", nil)

	uncorsHandler.ServeHTTP(responseWriter, request)

	assert.Equal(t, expectedBody, testutils.ReadBody(t, recorder))
}
