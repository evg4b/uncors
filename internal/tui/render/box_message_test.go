package render_test

import (
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/tui/render"
	"github.com/evg4b/uncors/testing/testutils"
)

var testCases = []struct {
	name    string
	message string
}{
	{
		name:    "render single line message",
		message: "test message",
	},
	{
		name:    "render multi line message",
		message: "test message\nsecond line",
	},
	{
		name:    "render message with empty lines",
		message: "test message\n\nsecond line",
	},
	{
		name:    "render empty message",
		message: "",
	},
	{
		name:    "render message with only empty lines",
		message: "\n\n\n",
	},
}

func runBoxTests(t *testing.T, write func(out *render.CliOutput, messages ...string)) {
	t.Helper()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			buffer := strings.Builder{}
			write(render.NewCliOutput(&buffer), testCase.message)
			testutils.MatchSnapshot(t, buffer.String())
		})
	}
}

func TestPrintWarnBox(t *testing.T) {
	runBoxTests(t, (*render.CliOutput).WarnBox)
}

func TestPrintInfoBox(t *testing.T) {
	runBoxTests(t, (*render.CliOutput).InfoBox)
}

func TestPrintErrorBox(t *testing.T) {
	runBoxTests(t, (*render.CliOutput).ErrorBox)
}
