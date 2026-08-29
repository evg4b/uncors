package infra_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

var errHandlerError = errors.New("handler error")

func TestHandlerFunc(t *testing.T) {
	t.Run("serves the request", func(t *testing.T) {
		const body = "served"

		handler := infra.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) error {
			fmt.Fprint(w, body)

			return nil
		})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, body, testutils.ReadBody(t, recorder))
	})

	t.Run("renders a returned error as an HTTP error", func(t *testing.T) {
		handler := infra.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) error {
			return errHandlerError
		})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Contains(t, testutils.ReadBody(t, recorder), errHandlerError.Error())
	})
}

func TestWithPrefix(t *testing.T) {
	t.Run("sets prefix in context", func(t *testing.T) {
		const prefix = "TEST"

		var capturedPrefix string

		handler := infra.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) error {
			if v, ok := r.Context().Value(contracts.PrefixKey).(string); ok {
				capturedPrefix = v
			}

			return nil
		})

		wrapped := infra.WithPrefix(prefix, handler)

		recorder := httptest.NewRecorder()
		writer := infra.NewResponseRecorder(recorder)
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		wrapped.ServeHTTP(writer, request)
		assert.Equal(t, prefix, capturedPrefix)
	})

	t.Run("calls prefix updater when present in context", func(t *testing.T) {
		const prefix = "UPDATED"

		updaterCalled := false
		updater := func(p string) {
			updaterCalled = true

			assert.Equal(t, prefix, p)
		}

		ctx := context.WithValue(t.Context(), contracts.PrefixUpdaterKey, func(s string) { updater(s) })
		request := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)

		handler := infra.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) error {
			return nil
		})

		wrapped := infra.WithPrefix(prefix, handler)

		recorder := httptest.NewRecorder()
		writer := infra.NewResponseRecorder(recorder)

		wrapped.ServeHTTP(writer, request)
		assert.True(t, updaterCalled)
	})
}

func TestPrefixedMiddleware(t *testing.T) {
	passthrough := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	t.Run("labels the requests the middleware passes through", func(t *testing.T) {
		const prefix = "PREFIXED"

		var capturedPrefix string

		next := infra.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) error {
			if v, ok := r.Context().Value(contracts.PrefixKey).(string); ok {
				capturedPrefix = v
			}

			return nil
		})

		prefixed := infra.NewPrefixedMiddleware(passthrough, prefix)

		recorder := httptest.NewRecorder()
		writer := infra.NewResponseRecorder(recorder)
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		prefixed(next).ServeHTTP(writer, request)
		assert.Equal(t, prefix, capturedPrefix)
	})

	t.Run("does not label requests the middleware answers itself", func(t *testing.T) {
		answering := func(http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusTeapot)
			})
		}

		prefixed := infra.NewPrefixedMiddleware(answering, "PREFIX")

		recorder := httptest.NewRecorder()
		writer := infra.NewResponseRecorder(recorder)
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		prefixed(mocks.FailNowHandlerMock(t)).ServeHTTP(writer, request)

		assert.Equal(t, http.StatusTeapot, recorder.Code)
	})
}
