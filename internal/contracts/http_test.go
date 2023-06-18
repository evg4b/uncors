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
	uncorsHandler := contracts.HandlerFunc(func(w *contracts.ResponseWriter, r *contracts.Request) {
		w.WriteHeader(http.StatusOK)
		sfmt.Fprint(w, expectedBody)
	})

	request := httptest.NewRequest(http.MethodGet, "/data", nil)
	httpHandler := contracts.CastToHTTPHandler(uncorsHandler)

	t.Run("cast correctly", func(t *testing.T) {
		recirder := httptest.NewRecorder()
		responceWriter := contracts.WrapResponseWriter(recirder)

		assert.NotPanics(t, func() {
			httpHandler.ServeHTTP(responceWriter, request)
			assert.Equal(t, expectedBody, testutils.ReadBody(t, recirder))
		})
	})

	t.Run("panit when request is not wrapped", func(t *testing.T) {
		recirder := httptest.NewRecorder()

		assert.PanicsWithValue(t, contracts.ErrResponceNotCasted, func() {
			httpHandler.ServeHTTP(recirder, request)
		})
	})
}

func TestHandlerFunc(t *testing.T) {
	const expectedBody = `{ "OK": true }`
	uncorsHandler := contracts.HandlerFunc(func(w *contracts.ResponseWriter, r *contracts.Request) {
		w.WriteHeader(http.StatusOK)
		sfmt.Fprint(w, expectedBody)
	})

	recirder := httptest.NewRecorder()
	responceWriter := contracts.WrapResponseWriter(recirder)
	request := httptest.NewRequest(http.MethodGet, "/data", nil)

	uncorsHandler.ServeHTTP(responceWriter, request)

	assert.Equal(t, expectedBody, testutils.ReadBody(t, recirder))
}
