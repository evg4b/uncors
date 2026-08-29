package version

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/evg4b/uncors/internal/tui"
	"github.com/hashicorp/go-version"
)

const lastVersionURL = "https://api.github.com/repos/evg4b/uncors/releases/latest"

type versionInfo struct {
	Version string `json:"tag_name"` // nolint: tagliatelle
}

func (checker *Checker) CheckNewVersion(ctx context.Context) {
	slog.Debug("checking for a new version")

	if checker.skip {
		slog.Debug("skipping the version check in debug mode")
		checker.output.Info("Skipping version check in debug mode")

		return
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, lastVersionURL, nil)
	if err != nil {
		slog.Debug("failed to build the version check request", "err", err)

		return
	}

	response, err := checker.http.Do(request)
	if err != nil {
		slog.Debug("version check request failed", "err", err)

		return
	}

	defer response.Body.Close()

	decoder := json.NewDecoder(response.Body)

	lastVersionInfo := versionInfo{}

	err = decoder.Decode(&lastVersionInfo)
	if err != nil {
		slog.Debug("failed to parse the version check response", "err", err)

		return
	}

	lastVersion, err := version.NewVersion(lastVersionInfo.Version)
	if err != nil {
		slog.Debug("failed to parse the latest version", "err", err)

		return
	}

	if lastVersion.GreaterThan(checker.currentVersion) {
		checker.output.Infof(tui.NewVersionIsAvailable, checker.currentVersion.String(), lastVersion.String())
		checker.output.Info("")
	} else {
		slog.Debug("version is up to date")
	}
}
