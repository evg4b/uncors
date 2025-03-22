package tui_test

import (
	"strings"
	"testing"

	"github.com/evg4b/uncors/testing/testutils"
	"github.com/gkampitakis/go-snaps/snaps"

	"github.com/evg4b/uncors/internal/tui"
)

func TestPrintWarningBox(t *testing.T) {
	tests := []struct {
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
	}
	for _, testCase := range tests {
		t.Run(testCase.name, testutils.WithTrueColor(func(t *testing.T) {
			buffer := strings.Builder{}

			tui.PrintWarningBox(&buffer, testCase.message)

			snaps.MatchSnapshot(t, buffer.String())
		}))
	}
}

func TestPrintInfoBox(t *testing.T) {
	tests := []struct {
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
	}
	for _, testCase := range tests {
		t.Run(testCase.name, testutils.WithTrueColor(func(t *testing.T) {
			buffer := strings.Builder{}

			tui.PrintInfoBox(&buffer, testCase.message)

			snaps.MatchSnapshot(t, buffer.String())
		}))
	}
}
