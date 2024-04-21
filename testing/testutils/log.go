package testutils

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/muesli/termenv"

	"github.com/charmbracelet/log"
)

func LogTest(action func(t *testing.T, output *bytes.Buffer)) func(t *testing.T) {
	buffer := &bytes.Buffer{}
	infra.ConfigureLogger()

	logger := log.Default()
	logger.SetOutput(buffer)
	logger.SetColorProfile(termenv.TrueColor)

	renderer := lipgloss.DefaultRenderer()
	renderer.SetColorProfile(termenv.TrueColor)

	return func(t *testing.T) {
		action(t, buffer)
	}
}

func UniqOutput(output *bytes.Buffer, action func(t *testing.T, output *bytes.Buffer)) func(t *testing.T) {
	return func(t *testing.T) {
		action(t, output)
		output.Reset()
	}
}
