//go:build !release

package version

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/contracts"
)

func CheckNewVersion(_ context.Context, _ contracts.HTTPClient, _ string) tea.Cmd {
	log.Debug("Check new version stub")

	return nil
}
