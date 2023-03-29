//go:build !release

package ui

import (
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/log"
)

func CheckLastVersion(_ contracts.HTTPClient, _ string) {
	log.Debug("Check new version stub")
}
