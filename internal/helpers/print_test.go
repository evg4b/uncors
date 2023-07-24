package helpers_test

import (
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testconstants"
	"github.com/stretchr/testify/assert"
)

const (
	rawPayload       = "test-data"
	fPayload         = "print formatted %s %d"
	fPayloadExpected = "print formatted string 555"
)

var fPayloadArgs = []any{"string", 555}

func TestFprint(t *testing.T) {
	t.Run("print correctly", func(t *testing.T) {
		writer := &strings.Builder{}

		helpers.Fprint(writer, rawPayload)

		assert.Equal(t, rawPayload, writer.String())
	})

	t.Run("panic on error", func(t *testing.T) {
		assert.Panics(t, func() {
			writer := mocks.NewWriterMock(t).
				WriteMock.Return(0, testconstants.ErrTest1)

			helpers.Fprint(writer, rawPayload)
		})
	})
}

func TestFprintf(t *testing.T) {
	t.Run("print correctly", func(t *testing.T) {
		writer := &strings.Builder{}

		helpers.Fprintf(writer, fPayload, fPayloadArgs...)

		assert.Equal(t, fPayloadExpected, writer.String())
	})

	t.Run("panic on error", func(t *testing.T) {
		assert.Panics(t, func() {
			writer := mocks.NewWriterMock(t).
				WriteMock.Return(0, testconstants.ErrTest1)

			helpers.Fprintf(writer, fPayload, fPayloadArgs...)
		})
	})
}

func TestFprintln(t *testing.T) {
	t.Run("print correctly", func(t *testing.T) {
		writer := &strings.Builder{}

		helpers.Fprintln(writer, rawPayload)

		assert.Equal(t, rawPayload+"\n", writer.String())
	})

	t.Run("panic on error", func(t *testing.T) {
		assert.Panics(t, func() {
			writer := mocks.NewWriterMock(t).
				WriteMock.Return(0, testconstants.ErrTest1)

			helpers.Fprintln(writer, rawPayload)
		})
	})
}

func TestSprintf(t *testing.T) {
	actual := helpers.Sprintf(fPayload, fPayloadArgs...)

	assert.Equal(t, fPayloadExpected, actual)
}
