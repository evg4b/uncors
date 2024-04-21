package tui_test

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer := strings.Builder{}
			lipgloss.DefaultRenderer().SetColorProfile(termenv.TrueColor)

			tui.PrintWarningBox(&buffer, tt.message)

			assert.Equal(t, tt.expected, buffer.String())
		})
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer := strings.Builder{}
			lipgloss.DefaultRenderer().SetColorProfile(termenv.TrueColor)

			tui.PrintInfoBox(&buffer, tt.message)

			assert.Equal(t, tt.expected, buffer.String())
		})
	}
}
