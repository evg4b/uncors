package version

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/hashicorp/go-version"
)

const lastVersionURL = "https://api.github.com/repos/evg4b/uncors/releases/latest"

type versionInfo struct {
	Version string `json:"tag_name"` // nolint: tagliatelle
}

// CheckNewVersion asks GitHub for the latest release. It returns that release
// and true when it is newer than the running version. Failures are logged and
// reported as no newer version: the check must never get in the user's way.
func (checker *Checker) CheckNewVersion(ctx context.Context) (string, bool) {
	slog.Debug("checking for a new version")

	if checker.skip {
		slog.Debug("skipping version check in debug mode")

		return "", false
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, lastVersionURL, nil)
	if err != nil {
		slog.Warn("failed to build the version check request", "error", err)

		return "", false
	}

	response, err := checker.http.Do(request)
	if err != nil {
		slog.Warn("version check request failed", "error", err)

		return "", false
	}

	defer helpers.CloseSafe(response.Body)

	decoder := json.NewDecoder(response.Body)

	lastVersionInfo := versionInfo{}

	err = decoder.Decode(&lastVersionInfo)
	if err != nil {
		slog.Warn("failed to parse the version check response", "error", err)

		return "", false
	}

	lastVersion, err := version.NewVersion(lastVersionInfo.Version)
	if err != nil {
		slog.Warn("failed to parse the latest version", "error", err)

		return "", false
	}

	if !lastVersion.GreaterThan(checker.currentVersion) {
		slog.Debug("version is up to date")

		return "", false
	}

	return lastVersion.String(), true
}
