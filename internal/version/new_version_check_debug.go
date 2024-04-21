//go:build !release

package version

import (
	"context"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/contracts"
)

func CheckNewVersion(_ context.Context, _ contracts.HTTPClient, _ string) {
	log.Debug("Check new version stub")
}
