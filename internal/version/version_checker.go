package version

import (
	"log/slog"
	"strings"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/hashicorp/go-version"
)

type Checker struct {
	http           contracts.HTTPClient
	currentVersion *version.Version
	skip           bool
}

type Option = func(*Checker)

func WithHTTPClient(client contracts.HTTPClient) Option {
	return func(checker *Checker) {
		checker.http = client
	}
}

func WithCurrentVersion(rawVersion string) Option {
	return func(checker *Checker) {
		if strings.EqualFold(rawVersion, "x.x.x") {
			checker.skip = true

			return
		}

		currentVersion, err := version.NewVersion(rawVersion)
		if err != nil {
			slog.Warn("skipping version check for an unparsable version", "version", rawVersion, "error", err)

			checker.skip = true

			return
		}

		checker.currentVersion = currentVersion
	}
}

func NewVersionChecker(options ...Option) *Checker {
	return helpers.ApplyOptions(&Checker{}, options)
}

func (checker *Checker) Current() string {
	if checker.currentVersion == nil {
		return ""
	}

	return checker.currentVersion.String()
}
