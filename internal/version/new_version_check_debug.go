//go:build !release

package version

import (
	"context"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/log"
)

func CheckNewVersion(_ context.Context, _ contracts.HTTPClient, _ string) {
	log.Default().Debug("Check new version stub")
}
