package infra_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseRecorder_StatusCode(t *testing.T) {
	t.Run("returns 0 by default", func(t *testing.T) {
		rec := infra.NewResponseRecorder(httptest.NewRecorder())

		assert.Equal(t, 0, rec.StatusCode())
	})

	t.Run("returns code set by WriteHeader", func(t *testing.T) {
		rec := infra.NewResponseRecorder(httptest.NewRecorder())
		rec.WriteHeader(http.StatusNotFound)

		assert.Equal(t, http.StatusNotFound, rec.StatusCode())
	})
}

func TestResponseRecorder_Write(t *testing.T) {
	t.Run("writes through to underlying writer", func(t *testing.T) {
		underlying := httptest.NewRecorder()
		rec := infra.NewResponseRecorder(underlying)

		_, err := rec.Write([]byte("hello"))
		require.NoError(t, err)

		assert.Equal(t, "hello", underlying.Body.String())
	})

	t.Run("buffers body and still writes through when capture is enabled", func(t *testing.T) {
		underlying := httptest.NewRecorder()
		rec := infra.NewResponseRecorder(underlying)
		rec.EnableBodyCapture(infra.DefaultCaptureLimit)

		_, err := rec.Write([]byte("buffered"))
		require.NoError(t, err)

		assert.Equal(t, "buffered", underlying.Body.String())
	})
}

func TestResponseRecorder_Captured(t *testing.T) {
	t.Run("returns correct status code", func(t *testing.T) {
		rec := infra.NewResponseRecorder(httptest.NewRecorder())
		rec.WriteHeader(http.StatusCreated)

		assert.Equal(t, http.StatusCreated, rec.Captured().StatusCode)
	})

	t.Run("normalises missing WriteHeader to 200", func(t *testing.T) {
		rec := infra.NewResponseRecorder(httptest.NewRecorder())

		assert.Equal(t, http.StatusOK, rec.Captured().StatusCode)
	})

	t.Run("body is nil when capture not enabled", func(t *testing.T) {
		rec := infra.NewResponseRecorder(httptest.NewRecorder())
		_, err := rec.Write([]byte("ignored"))
		require.NoError(t, err)

		assert.Nil(t, rec.Captured().Body)
	})

	t.Run("body is captured when EnableBodyCapture is called", func(t *testing.T) {
		rec := infra.NewResponseRecorder(httptest.NewRecorder())
		rec.EnableBodyCapture(infra.DefaultCaptureLimit)
		_, err := rec.Write([]byte("captured"))
		require.NoError(t, err)

		assert.Equal(t, []byte("captured"), rec.Captured().Body)
	})

	t.Run("second EnableBodyCapture call is a no-op", func(t *testing.T) {
		rec := infra.NewResponseRecorder(httptest.NewRecorder())
		rec.EnableBodyCapture(infra.DefaultCaptureLimit)
		rec.EnableBodyCapture(infra.DefaultCaptureLimit)
		_, err := rec.Write([]byte("once"))
		require.NoError(t, err)

		assert.Equal(t, []byte("once"), rec.Captured().Body)
	})

	t.Run("duration is non-zero", func(t *testing.T) {
		rec := infra.NewResponseRecorder(httptest.NewRecorder())

		time.Sleep(time.Millisecond)

		assert.Positive(t, rec.Captured().Duration)
	})
}

func TestResponseRecorder_ImplementsInterfaces(t *testing.T) {
	rec := infra.NewResponseRecorder(httptest.NewRecorder())

	t.Run("implements ResponseWriter", func(_ *testing.T) {
		var _ http.ResponseWriter = rec
	})

	t.Run("implements BodyCapturer", func(_ *testing.T) {
		var _ contracts.BodyCapturer = rec
	})
}

func TestResponseRecorderCapabilities(t *testing.T) {
	t.Run("forwards Flush to the underlying writer", func(t *testing.T) {
		underlying := &flushableWriter{ResponseRecorder: httptest.NewRecorder()}
		recorder := infra.NewResponseRecorder(underlying)

		flusher, ok := any(recorder).(http.Flusher)
		require.True(t, ok, "the recorder must be an http.Flusher, or nothing can stream")

		flusher.Flush()

		assert.True(t, underlying.flushed)
	})

	t.Run("is reachable through http.NewResponseController", func(t *testing.T) {
		underlying := &flushableWriter{ResponseRecorder: httptest.NewRecorder()}
		recorder := infra.NewResponseRecorder(underlying)

		require.NoError(t, http.NewResponseController(recorder).Flush())
		assert.True(t, underlying.flushed)
	})

	t.Run("reports that a plain writer cannot be hijacked", func(t *testing.T) {
		recorder := infra.NewResponseRecorder(httptest.NewRecorder())

		hijacker, ok := any(recorder).(http.Hijacker)
		require.True(t, ok)

		_, _, err := hijacker.Hijack()

		require.ErrorIs(t, err, http.ErrNotSupported)
	})
}

func TestResponseRecorderCaptureLimit(t *testing.T) {
	t.Run("keeps at most the requested number of bytes", func(t *testing.T) {
		const limit = 8

		underlying := httptest.NewRecorder()
		recorder := infra.NewResponseRecorder(underlying)
		recorder.EnableBodyCapture(limit)

		_, err := recorder.Write([]byte("0123456789abcdef"))
		require.NoError(t, err)

		capture := recorder.Captured()

		assert.Equal(t, "01234567", string(capture.Body))
		assert.True(t, capture.Truncated)
		assert.Equal(t, "0123456789abcdef", underlying.Body.String(), "the client still gets the whole body")
	})

	t.Run("keeps the larger limit when two consumers enable capture", func(t *testing.T) {
		recorder := infra.NewResponseRecorder(httptest.NewRecorder())
		recorder.EnableBodyCapture(4)
		recorder.EnableBodyCapture(16)

		_, err := recorder.Write([]byte("0123456789"))
		require.NoError(t, err)

		capture := recorder.Captured()

		assert.Equal(t, "0123456789", string(capture.Body))
		assert.False(t, capture.Truncated)
	})

	t.Run("does not buffer a streamed response", func(t *testing.T) {
		underlying := httptest.NewRecorder()
		recorder := infra.NewResponseRecorder(underlying)
		recorder.EnableBodyCapture(infra.DefaultCaptureLimit)

		recorder.Header().Set(headers.ContentType, "text/event-stream")
		recorder.WriteHeader(http.StatusOK)

		_, err := recorder.Write([]byte("data: tick\n\n"))
		require.NoError(t, err)

		assert.Empty(t, recorder.Captured().Body)
		assert.Equal(t, "data: tick\n\n", underlying.Body.String())
	})
}

type flushableWriter struct {
	*httptest.ResponseRecorder

	flushed bool
}

func (w *flushableWriter) Flush() {
	w.flushed = true
}
