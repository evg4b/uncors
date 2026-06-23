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
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareFunc(t *testing.T) {
	t.Run("Wrap calls handler with next", func(t *testing.T) {
		const body = "wrapped"

		nextCalled := false
		next := infra.HandlerFunc(func(w contracts.ResponseWriter, _ *contracts.Request) error {
			nextCalled = true
			fmt.Fprint(w, body)

			return nil
		})

		mw := infra.MiddlewareFunc(func(h contracts.Handler) contracts.Handler {
			return infra.HandlerFunc(func(w contracts.ResponseWriter, r *contracts.Request) error {
				return h.ServeHTTP(w, r)
			})
		})

		recorder := httptest.NewRecorder()
		writer := server.NewResponseRecorder(recorder)
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		handler := mw.Wrap(next)
		err := handler.ServeHTTP(writer, request)

		require.NoError(t, err)
		assert.True(t, nextCalled)
		assert.Equal(t, body, testutils.ReadBody(t, recorder))
	})
}

func TestCastToContractsHandler(t *testing.T) {
	t.Run("wraps http.Handler as contracts.Handler", func(t *testing.T) {
		const body = "contracts-handler"

		httpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, body)
		})

		contractsHandler := infra.CastToContractsHandler(httpHandler)

		recorder := httptest.NewRecorder()
		writer := server.NewResponseRecorder(recorder)
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		err := contractsHandler.ServeHTTP(writer, request)

		require.NoError(t, err)
		assert.Equal(t, body, testutils.ReadBody(t, recorder))
	})
}

func TestMddleware(t *testing.T) {
	t.Run("chains middleware and handler", func(t *testing.T) {
		const body = "chained"

		middlewareCalled := false
		mw := testMiddlewareFunc(func(w contracts.ResponseWriter, r *contracts.Request, next contracts.Next) error {
			middlewareCalled = true

			return next(w, r)
		})

		handler := infra.HandlerFunc(func(w contracts.ResponseWriter, _ *contracts.Request) error {
			fmt.Fprint(w, body)

			return nil
		})

		chained := infra.Mddleware(mw, handler)

		recorder := httptest.NewRecorder()
		writer := server.NewResponseRecorder(recorder)
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		err := chained.ServeHTTP(writer, request)

		require.NoError(t, err)
		assert.True(t, middlewareCalled)
		assert.Equal(t, body, testutils.ReadBody(t, recorder))
	})

	t.Run("propagates error from middleware", func(t *testing.T) {
		expectedErr := errors.New("middleware error")

		mw := testMiddlewareFunc(func(_ contracts.ResponseWriter, _ *contracts.Request, _ contracts.Next) error {
			return expectedErr
		})

		handler := infra.HandlerFunc(func(_ contracts.ResponseWriter, _ *contracts.Request) error {
			return nil
		})

		chained := infra.Mddleware(mw, handler)

		recorder := httptest.NewRecorder()
		writer := server.NewResponseRecorder(recorder)
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		err := chained.ServeHTTP(writer, request)

		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestWithPrefix(t *testing.T) {
	t.Run("sets prefix in context", func(t *testing.T) {
		const prefix = "TEST"

		var capturedPrefix string

		handler := infra.HandlerFunc(func(_ contracts.ResponseWriter, r *contracts.Request) error {
			if v, ok := r.Context().Value(contracts.PrefixKey).(string); ok {
				capturedPrefix = v
			}

			return nil
		})

		wrapped := infra.WithPrefix(prefix, handler)

		recorder := httptest.NewRecorder()
		writer := server.NewResponseRecorder(recorder)
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		err := wrapped.ServeHTTP(writer, request)

		require.NoError(t, err)
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

		handler := infra.HandlerFunc(func(_ contracts.ResponseWriter, _ *contracts.Request) error {
			return nil
		})

		wrapped := infra.WithPrefix(prefix, handler)

		recorder := httptest.NewRecorder()
		writer := server.NewResponseRecorder(recorder)

		err := wrapped.ServeHTTP(writer, request)

		require.NoError(t, err)
		assert.True(t, updaterCalled)
	})
}

func TestPrefixedMiddleware(t *testing.T) {
	t.Run("ServeHTTP sets prefix on next handler", func(t *testing.T) {
		const prefix = "PREFIXED"

		var capturedPrefix string

		innerNext := func(_ contracts.ResponseWriter, r *contracts.Request) error {
			if v, ok := r.Context().Value(contracts.PrefixKey).(string); ok {
				capturedPrefix = v
			}

			return nil
		}

		mw := testMiddlewareFunc(func(w contracts.ResponseWriter, r *contracts.Request, next contracts.Next) error {
			return next(w, r)
		})

		prefixed := infra.NewPrefixedMiddleware(mw, prefix)

		recorder := httptest.NewRecorder()
		writer := server.NewResponseRecorder(recorder)
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		err := prefixed.ServeHTTP(writer, request, innerNext)

		require.NoError(t, err)
		assert.Equal(t, prefix, capturedPrefix)
	})

	t.Run("propagates error from middleware", func(t *testing.T) {
		expectedErr := errors.New("middleware error")

		mw := testMiddlewareFunc(func(_ contracts.ResponseWriter, _ *contracts.Request, _ contracts.Next) error {
			return expectedErr
		})

		prefixed := infra.NewPrefixedMiddleware(mw, "PREFIX")

		recorder := httptest.NewRecorder()
		writer := server.NewResponseRecorder(recorder)
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		next := func(_ contracts.ResponseWriter, _ *contracts.Request) error { return nil }
		err := prefixed.ServeHTTP(writer, request, next)

		assert.ErrorIs(t, err, expectedErr)
	})
}

type testMiddlewareFunc func(contracts.ResponseWriter, *contracts.Request, contracts.Next) error

func (f testMiddlewareFunc) ServeHTTP(w contracts.ResponseWriter, r *contracts.Request, next contracts.Next) error {
	return f(w, r, next)
}
