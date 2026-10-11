package tui_test

import (
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/tui"
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

func runBoxTests(t *testing.T, write func(out *tui.CliOutput, messages ...string)) {
	t.Helper()

	for _, testCase := range testCases {
		t.Run(testCase.name, testutils.WithTrueColor(func(t *testing.T) {
			buffer := strings.Builder{}
			write(tui.NewCliOutput(&buffer), testCase.message)
			testutils.MatchSnapshot(t, buffer.String())
		}))
	}
}

func TestPrintWarnBox(t *testing.T) {
	runBoxTests(t, (*tui.CliOutput).WarnBox)
}

func TestPrintInfoBox(t *testing.T) {
	runBoxTests(t, (*tui.CliOutput).InfoBox)
}

func TestPrintErrorBox(t *testing.T) {
	runBoxTests(t, (*tui.CliOutput).ErrorBox)
}
