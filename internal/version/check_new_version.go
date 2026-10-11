package version

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/hashicorp/go-version"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const lastVersionURL = "https://api.github.com/repos/evg4b/uncors/releases/latest"

type versionInfo struct {
	Version string `json:"tag_name"` // nolint: tagliatelle
}

func (checker *Checker) CheckNewVersion(ctx context.Context) (string, bool) {
	ctx, span := telemetry.StartKind(ctx, "version.check", trace.SpanKindClient)
	defer span.End()

	if checker.skip {
		span.SetAttributes(attribute.Bool("uncors.version.skipped", true))
		telemetry.Debug(ctx, "skipping version check for a development build")

		return "", false
	}

	latest, err := checker.fetchLatest(ctx)
	if err != nil {
		telemetry.Fail(span, err)
		telemetry.Debug(ctx, "version check failed", slog.String("error", err.Error()))

		return "", false
	}

	span.SetAttributes(
		attribute.String("uncors.version.current", checker.currentVersion.String()),
		attribute.String("uncors.version.latest", latest.String()),
	)

	if !latest.GreaterThan(checker.currentVersion) {
		return "", false
	}

	return latest.String(), true
}

func (checker *Checker) fetchLatest(ctx context.Context) (*version.Version, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, lastVersionURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build the version check request: %w", err)
	}

	response, err := checker.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("version check request failed: %w", err)
	}

	defer helpers.CloseSafe(response.Body)

	lastVersionInfo := versionInfo{}

	err = json.NewDecoder(response.Body).Decode(&lastVersionInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the version check response: %w", err)
	}

	lastVersion, err := version.NewVersion(lastVersionInfo.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the latest version: %w", err)
	}

	return lastVersion, nil
}
