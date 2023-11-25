package helpers_test

import (
	"io"
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

func TestFPrint(t *testing.T) {
	simplePrintTest(t, rawPayload, func(writer io.Writer) {
		helpers.FPrint(writer, rawPayload)
	})
}

func TestFprintf(t *testing.T) {
	simplePrintTest(t, fPayloadExpected, func(writer io.Writer) {
		helpers.FPrintf(writer, fPayload, fPayloadArgs...)
	})
}

func TestFPrintln(t *testing.T) {
	simplePrintTest(t, rawPayload+"\n", func(writer io.Writer) {
		helpers.FPrintln(writer, rawPayload)
	})
}

func simplePrintTest(t *testing.T, expected string, action func(writer io.Writer)) {
	t.Run("print correctly", func(t *testing.T) {
		writer := &strings.Builder{}

		action(writer)

		assert.Equal(t, expected, writer.String())
	})

	t.Run("panic on error", func(t *testing.T) {
		assert.Panics(t, func() {
			writer := mocks.NewWriterMock(t).
				WriteMock.Return(0, testconstants.ErrTest1)

			action(writer)
		})
	})
}

func TestSprintf(t *testing.T) {
	actual := helpers.Sprintf(fPayload, fPayloadArgs...)

	assert.Equal(t, fPayloadExpected, actual)
}
