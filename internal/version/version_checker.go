package version

import (
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/hashicorp/go-version"
)

type Checker struct {
	output         contracts.Output
	http           contracts.HTTPClient
	currentVersion *version.Version
	skip           bool
}

type Option = func(*Checker)

func WithOutput(output contracts.Output) Option {
	return func(checker *Checker) {
		checker.output = output
	}
}

func WithHTTPClient(client contracts.HTTPClient) Option {
	return func(checker *Checker) {
		checker.http = client
	}
}

func WithCurrentVersion(rawVersion string) Option {
	return func(checker *Checker) {
		currentVersion, err := version.NewVersion(rawVersion)
		if err != nil {
			panic(err)
		}

		checker.currentVersion = currentVersion
	}
}

func WithSkip() Option {
	return func(checker *Checker) {
		checker.skip = true
	}
}

func NewVersionChecker(options ...Option) *Checker {
	return helpers.ApplyOptions(&Checker{}, options)
}
