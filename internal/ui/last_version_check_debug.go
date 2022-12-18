//go:build !release

package ui

import (
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/log"
)

func CheckLastVersion(client contracts.HTTPClient, reCurrentVersion string) {
	log.Debug("Check new version stub")
}
