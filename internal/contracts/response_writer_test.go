package contracts_test

import (
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestResponseWriterWrapper(t *testing.T) {
	const expectedValue = `{ "status": "ok" }`
	const expectedCode = 201

	recorder := httptest.NewRecorder()
	writer := contracts.WrapResponseWriter(recorder)

	writer.WriteHeader(expectedCode)
	helpers.Fprint(writer, expectedValue)

	t.Run("save status code", func(t *testing.T) {
		assert.Equal(t, expectedCode, writer.StatusCode())
	})

	t.Run("write body", func(t *testing.T) {
		assert.Equal(t, expectedValue, testutils.ReadBody(t, recorder))
	})
}
