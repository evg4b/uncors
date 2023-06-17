package static_test

import (
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/handler/static"
	"github.com/evg4b/uncors/internal/sfmt"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestResponseWriterWrapper(t *testing.T) {
	const expectedValye = `{ "status": "ok" }`
	const expectedCode = 201

	recorder := httptest.NewRecorder()
	writer := static.WrapResponseWriter(recorder)

	writer.WriteHeader(expectedCode)
	sfmt.Fprint(writer, expectedValye)

	t.Run("save status code", func(t *testing.T) {
		assert.Equal(t, expectedCode, writer.StatusCode)
	})

	t.Run("write body", func(t *testing.T) {
		assert.Equal(t, expectedValye, testutils.ReadBody(t, recorder))
	})
}
