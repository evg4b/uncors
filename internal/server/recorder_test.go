package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseRecorder_StatusCode(t *testing.T) {
	t.Run("returns 0 by default", func(t *testing.T) {
		rec := server.NewResponseRecorder(httptest.NewRecorder())

		assert.Equal(t, 0, rec.StatusCode())
	})

	t.Run("returns code set by WriteHeader", func(t *testing.T) {
		rec := server.NewResponseRecorder(httptest.NewRecorder())
		rec.WriteHeader(http.StatusNotFound)

		assert.Equal(t, http.StatusNotFound, rec.StatusCode())
	})
}

func TestResponseRecorder_Write(t *testing.T) {
	t.Run("writes through to underlying writer", func(t *testing.T) {
		underlying := httptest.NewRecorder()
		rec := server.NewResponseRecorder(underlying)

		_, err := rec.Write([]byte("hello"))
		require.NoError(t, err)

		assert.Equal(t, "hello", underlying.Body.String())
	})

	t.Run("buffers body and still writes through when capture is enabled", func(t *testing.T) {
		underlying := httptest.NewRecorder()
		rec := server.NewResponseRecorder(underlying)
		rec.EnableBodyCapture()

		_, err := rec.Write([]byte("buffered"))
		require.NoError(t, err)

		assert.Equal(t, "buffered", underlying.Body.String())
	})
}

func TestResponseRecorder_Captured(t *testing.T) {
	t.Run("returns correct status code", func(t *testing.T) {
		rec := server.NewResponseRecorder(httptest.NewRecorder())
		rec.WriteHeader(http.StatusCreated)

		assert.Equal(t, http.StatusCreated, rec.Captured().StatusCode)
	})

	t.Run("normalises missing WriteHeader to 200", func(t *testing.T) {
		rec := server.NewResponseRecorder(httptest.NewRecorder())

		assert.Equal(t, http.StatusOK, rec.Captured().StatusCode)
	})

	t.Run("body is nil when capture not enabled", func(t *testing.T) {
		rec := server.NewResponseRecorder(httptest.NewRecorder())
		_, _ = rec.Write([]byte("ignored"))

		assert.Nil(t, rec.Captured().Body)
	})

	t.Run("body is captured when EnableBodyCapture is called", func(t *testing.T) {
		rec := server.NewResponseRecorder(httptest.NewRecorder())
		rec.EnableBodyCapture()
		_, _ = rec.Write([]byte("captured"))

		assert.Equal(t, []byte("captured"), rec.Captured().Body)
	})

	t.Run("second EnableBodyCapture call is a no-op", func(t *testing.T) {
		rec := server.NewResponseRecorder(httptest.NewRecorder())
		rec.EnableBodyCapture()
		rec.EnableBodyCapture()
		_, _ = rec.Write([]byte("once"))

		assert.Equal(t, []byte("once"), rec.Captured().Body)
	})

	t.Run("duration is non-zero", func(t *testing.T) {
		rec := server.NewResponseRecorder(httptest.NewRecorder())
		time.Sleep(time.Millisecond)

		assert.Positive(t, rec.Captured().Duration)
	})
}

func TestResponseRecorder_ImplementsInterfaces(t *testing.T) {
	rec := server.NewResponseRecorder(httptest.NewRecorder())

	t.Run("implements contracts.ResponseWriter", func(t *testing.T) {
		var _ contracts.ResponseWriter = rec
	})

	t.Run("implements contracts.BodyCapturer", func(t *testing.T) {
		var _ contracts.BodyCapturer = rec
	})
}
