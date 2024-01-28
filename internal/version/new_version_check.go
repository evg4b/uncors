//go:build release

package version

import (
	"context"
	"encoding/json"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/evg4b/uncors/internal/uncors"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/hashicorp/go-version"
)

const lastVersionURL = "https://api.github.com/repos/evg4b/uncors/releases/latest"

type versionInfo struct {
	Version string `json:"tag_name"`
}

func CheckNewVersion(ctx context.Context, client contracts.HTTPClient, rawCurrentVersion string) tea.Cmd {
	log.Debug("Checking new version")

	currentVersion, err := version.NewVersion(rawCurrentVersion)
	if err != nil {
		log.Debugf("failed to parse current version: %v", err)

		return nil
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, lastVersionURL, nil)
	if err != nil {
		log.Debugf("failed to generate new version check request: %v", err)

		return nil
	}

	response, err := client.Do(request)
	if err != nil {
		log.Debugf("http error occupied: %v", err)

		return nil
	}

	defer helpers.CloseSafe(response.Body)
	decoder := json.NewDecoder(response.Body)

	lastVersionInfo := versionInfo{}
	err = decoder.Decode(&lastVersionInfo)
	if err != nil {
		log.Debugf("failed to parse last version response: %v", err)

		return nil
	}

	lastVersion, err := version.NewVersion(lastVersionInfo.Version)
	if err != nil {
		log.Debugf("failed to parse last version: %v", err)

		return nil
	}

	if lastVersion.GreaterThan(currentVersion) {
		return tea.Printf(uncors.NewVersionIsAvailable, currentVersion.String(), lastVersion.String())
	}

	log.Debug("Version is up to date")

	return nil
}
