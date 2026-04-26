//go:build release

package version

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/log"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/hashicorp/go-version"
)

const lastVersionURL = "https://api.github.com/repos/evg4b/uncors/releases/latest"

type versionInfo struct {
	Version string `json:"tag_name"`
}

func CheckNewVersion(ctx context.Context, client contracts.HTTPClient, rawCurrentVersion string) {
	logger := log.Default()

	logger.Debug("Checking new version")

	currentVersion, err := version.NewVersion(rawCurrentVersion)
	if err != nil {
		logger.Debugf("failed to parse current version: %v", err)

		return
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, lastVersionURL, nil)
	if err != nil {
		logger.Debugf("failed to generate new version check request: %v", err)

		return
	}

	response, err := client.Do(request)
	if err != nil {
		logger.Debugf("http error occurred: %v", err)

		return
	}

	defer helpers.CloseSafe(response.Body)
	decoder := json.NewDecoder(response.Body)

	lastVersionInfo := versionInfo{}
	err = decoder.Decode(&lastVersionInfo)
	if err != nil {
		logger.Debugf("failed to parse last version response: %v", err)

		return
	}

	lastVersion, err := version.NewVersion(lastVersionInfo.Version)
	if err != nil {
		logger.Debugf("failed to parse last version: %v", err)

		return
	}

	if lastVersion.GreaterThan(currentVersion) {
		logger.Infof(uncors.NewVersionIsAvailable, currentVersion.String(), lastVersion.String())
		logger.Info("")
	} else {
		logger.Debug("Version is up to date")
	}
}
