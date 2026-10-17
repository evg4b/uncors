package server_test

import (
	"bytes"
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
		_, err := rec.Write([]byte("ignored"))
		require.NoError(t, err)

		assert.Nil(t, rec.Captured().Body)
	})

	t.Run("body is captured when EnableBodyCapture is called", func(t *testing.T) {
		rec := server.NewResponseRecorder(httptest.NewRecorder())
		rec.EnableBodyCapture()
		_, err := rec.Write([]byte("captured"))
		require.NoError(t, err)

		assert.Equal(t, []byte("captured"), rec.Captured().Body)
	})

	t.Run("second EnableBodyCapture call is a no-op", func(t *testing.T) {
		rec := server.NewResponseRecorder(httptest.NewRecorder())
		rec.EnableBodyCapture()
		rec.EnableBodyCapture()
		_, err := rec.Write([]byte("once"))
		require.NoError(t, err)

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

	t.Run("implements ResponseWriter", func(_ *testing.T) {
		var _ contracts.ResponseWriter = rec
	})

	t.Run("implements BodyCapturer", func(_ *testing.T) {
		var _ contracts.BodyCapturer = rec
	})
}

func TestResponseRecorder_CaptureLimit(t *testing.T) {
	t.Run("captures a body up to the limit", func(t *testing.T) {
		rec := server.NewResponseRecorder(httptest.NewRecorder())
		rec.EnableBodyCapture()

		_, err := rec.Write(bytes.Repeat([]byte("x"), server.MaxCapturedBody))
		require.NoError(t, err)

		captured := rec.Captured()
		assert.Len(t, captured.Body, server.MaxCapturedBody)
		assert.False(t, captured.Truncated)
	})

	t.Run("stops capturing and marks the capture truncated over the limit", func(t *testing.T) {
		underlying := httptest.NewRecorder()
		rec := server.NewResponseRecorder(underlying)
		rec.EnableBodyCapture()

		chunk := bytes.Repeat([]byte("x"), server.MaxCapturedBody/2+1)
		for range 3 {
			_, err := rec.Write(chunk)
			require.NoError(t, err)
		}

		captured := rec.Captured()
		assert.True(t, captured.Truncated)
		assert.Len(t, captured.Body, server.MaxCapturedBody)
		assert.Equal(t, 3*len(chunk), underlying.Body.Len())
		assert.Equal(t, int64(3*len(chunk)), rec.Written())
	})
}

type flushCounter struct {
	http.ResponseWriter

	flushes int
}

func (f *flushCounter) Flush() {
	f.flushes++
}

type deadlineWriter struct {
	http.ResponseWriter

	deadline time.Time
}

func (d *deadlineWriter) SetWriteDeadline(deadline time.Time) error {
	d.deadline = deadline

	return nil
}

type plainWriter struct {
	http.ResponseWriter
}

func TestResponseRecorder_Flush(t *testing.T) {
	t.Run("flushes the underlying writer", func(t *testing.T) {
		underlying := &flushCounter{ResponseWriter: httptest.NewRecorder()}
		rec := server.NewResponseRecorder(underlying)

		var flusher http.Flusher = rec

		flusher.Flush()
		require.NoError(t, http.NewResponseController(rec).Flush())

		assert.Equal(t, 2, underlying.flushes)
	})

	t.Run("reports that the underlying writer cannot flush", func(t *testing.T) {
		rec := server.NewResponseRecorder(plainWriter{ResponseWriter: httptest.NewRecorder()})

		assert.NotPanics(t, rec.Flush)
		require.ErrorIs(t, http.NewResponseController(rec).Flush(), http.ErrNotSupported)
	})

	t.Run("unwraps to the underlying writer", func(t *testing.T) {
		underlying := httptest.NewRecorder()
		rec := server.NewResponseRecorder(underlying)

		assert.Same(t, underlying, rec.Unwrap())
	})

	t.Run("reaches the underlying writer through a response controller", func(t *testing.T) {
		underlying := &deadlineWriter{ResponseWriter: httptest.NewRecorder()}
		rec := server.NewResponseRecorder(underlying)
		deadline := time.Now().Add(time.Minute)

		require.NoError(t, http.NewResponseController(rec).SetWriteDeadline(deadline))

		assert.Equal(t, deadline, underlying.deadline)
	})
}
