package sfmt_test

import (
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/sfmt"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testconstants"
	"github.com/stretchr/testify/assert"
)

const (
	rawPayload       = "test-data"
	fPayload         = "pint f %s %d"
	fPayloadExpected = "pint f demo 555"
)

var fPayloadArgs = []any{"demo", 555}

func TestFprint(t *testing.T) {
	t.Run("print correctly", func(t *testing.T) {
		writer := &strings.Builder{}

		sfmt.Fprint(writer, rawPayload)

		assert.Equal(t, rawPayload, writer.String())
	})

	t.Run("panic on error", func(t *testing.T) {
		assert.Panics(t, func() {
			writer := mocks.NewWriterMock(t).
				WriteMock.Return(0, testconstants.ErrTest1)

			sfmt.Fprint(writer, rawPayload)
		})
	})
}

func TestFprintf(t *testing.T) {
	t.Run("print correctly", func(t *testing.T) {
		writer := &strings.Builder{}

		sfmt.Fprintf(writer, fPayload, fPayloadArgs...)

		assert.Equal(t, fPayloadExpected, writer.String())
	})

	t.Run("panic on error", func(t *testing.T) {
		assert.Panics(t, func() {
			writer := mocks.NewWriterMock(t).
				WriteMock.Return(0, testconstants.ErrTest1)

			sfmt.Fprintf(writer, fPayload, fPayloadArgs...)
		})
	})
}

func TestFprintln(t *testing.T) {
	t.Run("print correctly", func(t *testing.T) {
		writer := &strings.Builder{}

		sfmt.Fprintln(writer, rawPayload)

		assert.Equal(t, rawPayload+"\n", writer.String())
	})

	t.Run("panic on error", func(t *testing.T) {
		assert.Panics(t, func() {
			writer := mocks.NewWriterMock(t).
				WriteMock.Return(0, testconstants.ErrTest1)

			sfmt.Fprintln(writer, rawPayload)
		})
	})
}

func TestSprintf(t *testing.T) {
	actual := sfmt.Sprintf(fPayload, fPayloadArgs...)

	assert.Equal(t, fPayloadExpected, actual)
}
