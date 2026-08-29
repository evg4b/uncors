package infra_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/infra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A plain http.ResponseWriter has no body capture; the pipeline must degrade
// instead of asserting that a recorder is present.
func TestCaptureFrom(t *testing.T) {
	t.Run("finds the recorder installed by the pipeline", func(t *testing.T) {
		recorder := infra.NewResponseRecorder(httptest.NewRecorder())

		capturer, ok := infra.CaptureFrom(recorder)

		require.True(t, ok)
		assert.Same(t, recorder, capturer)
	})

	t.Run("looks through writers layered on top of the recorder", func(t *testing.T) {
		recorder := infra.NewResponseRecorder(httptest.NewRecorder())

		capturer, ok := infra.CaptureFrom(&wrappedWriter{ResponseWriter: recorder})

		require.True(t, ok)
		assert.Same(t, recorder, capturer)
	})

	t.Run("reports a plain writer instead of panicking", func(t *testing.T) {
		capturer, ok := infra.CaptureFrom(httptest.NewRecorder())

		assert.False(t, ok)
		assert.Nil(t, capturer)
	})
}

type wrappedWriter struct {
	http.ResponseWriter
}

func (w *wrappedWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
