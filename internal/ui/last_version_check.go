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

func CheckLastVersion(client contracts.HTTPClient, rewCurrectVersion string) {
	log.Debug("Checking new version")

	currectVersion, err := version.NewVersion(rewCurrectVersion)
	if err != nil {
		log.Debugf("failed to parse currect version: %v", err)

		return
	}

	url, _ := url.Parse(lastVersionUrl)
	response, err := client.Do(&http.Request{URL: url})
	if err != nil {
		log.Debugf("http error ocupted: %v", err)

		return
	}

	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)

	lastVersionInfo := versionInfo{}
	err = decoder.Decode(&lastVersionInfo)
	if err != nil {
		log.Debugf("failed to parse last version respoce: %v", err)

		return
	}

	lastVersion, err := version.NewVersion(lastVersionInfo.Version)
	if err != nil {
		log.Debugf("failed to parse last version: %v", err)

		return
	}

	if lastVersion.GreaterThan(currectVersion) {
		log.Infof(NewVersionIsAvailable, currectVersion.String(), lastVersion.String())
		log.Print("\n")
	} else {
		log.Debug("Version is up to date")
	}
}
