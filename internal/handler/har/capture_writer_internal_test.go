package har

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/stretchr/testify/assert"
)

func TestCaptureWriter_StatusCode(t *testing.T) {
	t.Run("returns 200 by default", func(t *testing.T) {
		rec := httptest.NewRecorder()
		cw := newCaptureWriter(contracts.WrapResponseWriter(rec))

		assert.Equal(t, http.StatusOK, cw.StatusCode())
	})

	t.Run("returns code set by WriteHeader", func(t *testing.T) {
		rec := httptest.NewRecorder()
		cw := newCaptureWriter(contracts.WrapResponseWriter(rec))

		cw.WriteHeader(http.StatusNotFound)

		assert.Equal(t, http.StatusNotFound, cw.StatusCode())
	})
}
