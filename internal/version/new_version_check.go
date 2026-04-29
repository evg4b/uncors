//go:build release

package version

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/hashicorp/go-version"
)

const lastVersionURL = "https://api.github.com/repos/evg4b/uncors/releases/latest"

type versionInfo struct {
	Version string `json:"tag_name"`
}

func CheckNewVersion(ctx context.Context, output contracts.Output, client contracts.HTTPClient, rawCurrentVersion string) {
	log.Print("Checking new version")

	currentVersion, err := version.NewVersion(rawCurrentVersion)
	if err != nil {
		log.Printf("failed to parse current version: %v", err)

		return
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, lastVersionURL, nil)
	if err != nil {
		log.Printf("failed to generate new version check request: %v", err)

		return
	}

	response, err := client.Do(request)
	if err != nil {
		log.Printf("http error occurred: %v", err)

		return
	}

	defer helpers.CloseSafe(response.Body)
	decoder := json.NewDecoder(response.Body)

	lastVersionInfo := versionInfo{}
	err = decoder.Decode(&lastVersionInfo)
	if err != nil {
		log.Printf("failed to parse last version response: %v", err)

		return
	}

	lastVersion, err := version.NewVersion(lastVersionInfo.Version)
	if err != nil {
		log.Printf("failed to parse last version: %v", err)

		return
	}

	if lastVersion.GreaterThan(currentVersion) {
		output.Infof(uncors.NewVersionIsAvailable, currentVersion.String(), lastVersion.String())
		output.Info("")
	} else {
		log.Print("Version is up to date")
	}
}
