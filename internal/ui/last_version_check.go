//go:build release

package ui

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/log"
	version "github.com/hashicorp/go-version"
)

const lastVersionUrl = "https://api.github.com/repos/evg4b/uncors/releases/latest"

type versionInfo struct {
	Version string `json:"tag_name"`
}

func CheckLastVersion(client contracts.HTTPClient, reCurrentVersion string) {
	log.Debug("Checking new version")

	currentVersion, err := version.NewVersion(reCurrentVersion)
	if err != nil {
		log.Debugf("failed to parse current version: %v", err)

		return
	}

	url, _ := url.Parse(lastVersionUrl)
	response, err := client.Do(&http.Request{URL: url})
	if err != nil {
		log.Debugf("http error occupied: %v", err)

		return
	}

	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)

	lastVersionInfo := versionInfo{}
	err = decoder.Decode(&lastVersionInfo)
	if err != nil {
		log.Debugf("failed to parse last version response: %v", err)

		return
	}

	lastVersion, err := version.NewVersion(lastVersionInfo.Version)
	if err != nil {
		log.Debugf("failed to parse last version: %v", err)

		return
	}

	if lastVersion.GreaterThan(currentVersion) {
		log.Infof(NewVersionIsAvailable, currentVersion.String(), lastVersion.String())
		log.Print("\n")
	} else {
		log.Debug("Version is up to date")
	}
}
