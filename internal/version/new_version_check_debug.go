//go:build !release

package version

import (
	"context"
	"log"

	"github.com/evg4b/uncors/internal/contracts"
)

func CheckNewVersion(_ context.Context, _ contracts.Output, _ contracts.HTTPClient, _ string) {
	log.Print("Check new version stub")
}
