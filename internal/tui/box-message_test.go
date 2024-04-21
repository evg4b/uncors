package tui_test

import (
	"strings"
	"testing"

	"github.com/evg4b/uncors/testing/testutils"

	"github.com/evg4b/uncors/internal/tui"
	"github.com/stretchr/testify/assert"
)

func TestPrintWarningBox(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:    "render single line message",
			message: "test message",
			expected: "\x1b[48;2;255;211;0m \x1b[0m\x1b[38;2;0;0;0;48;2;255;211;" +
				"0mWARN \x1b[0m\x1b[48;2;255;211;0m \x1b[0m test message\n",
		},
		{
			name:    "render multi line message",
			message: "test message\nsecond line",
			expected: "\x1b[48;2;255;211;0m \x1b[0m\x1b[38;2;0;0;0;48;2;255;211;" +
				"0mWARN \x1b[0m\x1b[48;2;255;211;0m \x1b[0m test message\n\x1b[48;2;255" +
				";211;0m \x1b[0m\x1b[38;2;0;0;0;48;2;255;211;0m\x1b[0m\x1b[48;2;255;" +
				"211;0m \x1b[0m\x1b[48;2;255;211;0m     \x1b[0m second line \n",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, testutils.WithTrueColor(func(t *testing.T) {
			buffer := strings.Builder{}

			tui.PrintWarningBox(&buffer, testCase.message)

			assert.Equal(t, testCase.expected, buffer.String())
		}))
	}
}

func TestPrintInfoBox(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:    "render single line message",
			message: "test message",
			expected: "\x1b[48;2;0;113;206m \x1b[0m\x1b[38;2;0;0;0;48;2;0;113;" +
				"206mINFO \x1b[0m\x1b[48;2;0;113;206m \x1b[0m test message\n",
		},
		{
			name:    "render multi line message",
			message: "test message\nsecond line",
			expected: "\x1b[48;2;0;113;206m \x1b[0m\x1b[38;2;0;0;0;48;2;0;113;" +
				"206mINFO \x1b[0m\x1b[48;2;0;113;206m \x1b[0m test message\n\x1b[48;2;0;" +
				"113;206m \x1b[0m\x1b[38;2;0;0;0;48;2;0;113;206m\x1b[0m\x1b[48;2;0;113;" +
				"206m \x1b[0m\x1b[48;2;0;113;206m     \x1b[0m second line \n",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, testutils.WithTrueColor(func(t *testing.T) {
			buffer := strings.Builder{}

			tui.PrintInfoBox(&buffer, testCase.message)

			assert.Equal(t, testCase.expected, buffer.String())
		}))
	}
}
