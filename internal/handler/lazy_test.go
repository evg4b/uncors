package handler_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLazyHandler(t *testing.T) {
	t.Run("initialises handler on first request only", func(t *testing.T) {
		var callCount atomic.Int32

		lazyH := handler.LazyHandler(func() contracts.Handler {
			callCount.Add(1)

			return contracts.HandlerFunc(func(w contracts.ResponseWriter, _ *contracts.Request) {
				w.WriteHeader(http.StatusOK)
			})
		})

		for range 3 {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			lazyH.ServeHTTP(contracts.WrapResponseWriter(recorder), request)
		}

		assert.Equal(t, int32(1), callCount.Load())
	})

	t.Run("delegates requests to the created handler", func(t *testing.T) {
		const expectedBody = "lazy response"

		lazyH := handler.LazyHandler(func() contracts.Handler {
			return contracts.HandlerFunc(func(w contracts.ResponseWriter, _ *contracts.Request) {
				w.WriteHeader(http.StatusCreated)
				_, err := fmt.Fprint(w, expectedBody)
				require.NoError(t, err)
			})
		})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		lazyH.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

		assert.Equal(t, http.StatusCreated, recorder.Code)
		assert.Equal(t, expectedBody, recorder.Body.String())
	})
}

func TestLazyMiddleware(t *testing.T) {
	t.Run("initialises middleware on first request only", func(t *testing.T) {
		var callCount atomic.Int32

		next := contracts.HandlerFunc(func(w contracts.ResponseWriter, _ *contracts.Request) {
			w.WriteHeader(http.StatusOK)
		})

		lazyM := handler.LazyMiddleware(func() contracts.Middleware {
			callCount.Add(1)

			return &trackingMiddleware{label: "m", tracker: &[]string{}}
		})

		wrapped := lazyM.Wrap(next)

		for range 3 {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			wrapped.ServeHTTP(contracts.WrapResponseWriter(recorder), request)
		}

		assert.Equal(t, int32(1), callCount.Load())
	})

	t.Run("delegates requests through the created middleware to next handler", func(t *testing.T) {
		var calls []string

		next := contracts.HandlerFunc(func(w contracts.ResponseWriter, _ *contracts.Request) {
			calls = append(calls, "next")

			w.WriteHeader(http.StatusOK)
		})

		lazyM := handler.LazyMiddleware(func() contracts.Middleware {
			return &trackingMiddleware{label: "lazy", tracker: &calls}
		})

		wrapped := lazyM.Wrap(next)
		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		wrapped.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

		assert.Equal(t, []string{"lazy", "next"}, calls)
	})
}

type trackingMiddleware struct {
	label   string
	tracker *[]string
}

func (m *trackingMiddleware) Wrap(next contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(w contracts.ResponseWriter, r *contracts.Request) {
		*m.tracker = append(*m.tracker, m.label)

		next.ServeHTTP(w, r)
	})
}
