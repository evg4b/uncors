package testutils

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/muesli/termenv"
)

func WithTrueColor(action func(t *testing.T)) func(t *testing.T) {
	return func(t *testing.T) {
		logger := log.Default()
		renderer := lipgloss.DefaultRenderer()

		logger.SetColorProfile(termenv.TrueColor)
		renderer.SetColorProfile(termenv.TrueColor)

		t.Cleanup(func() {
			profile := termenv.ColorProfile()
			logger.SetColorProfile(profile)
			renderer.SetColorProfile(profile)
		})

		action(t)
	}
}
