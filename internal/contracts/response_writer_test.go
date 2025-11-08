package contracts_test

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestResponseWriterWrapper(t *testing.T) {
	const (
		expectedValue = `{ "status": "ok" }`
		expectedCode  = 201
	)

	recorder := httptest.NewRecorder()
	writer := contracts.WrapResponseWriter(recorder)

	writer.WriteHeader(expectedCode)
	fmt.Fprint(writer, expectedValue)

	t.Run("save status code", func(t *testing.T) {
		assert.Equal(t, expectedCode, writer.StatusCode())
	})

	t.Run("write body", func(t *testing.T) {
		assert.JSONEq(t, expectedValue, testutils.ReadBody(t, recorder))
	})
}
