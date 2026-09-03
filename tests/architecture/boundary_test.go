// Package architecture_test enforces the service/TUI boundary in CI.
//
// The point of separating the service from Bubble Tea is easy to state and
// easy to erode: one convenient import puts terminal concerns back into the
// application. These tests fail the build when that happens.
package architecture_test

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serviceSide is every package that must run correctly with no TUI attached.
var serviceSide = []string{
	"github.com/evg4b/uncors/internal/app/...",
	"github.com/evg4b/uncors/internal/di/...",
	"github.com/evg4b/uncors/internal/server/...",
	"github.com/evg4b/uncors/internal/handler/...",
	"github.com/evg4b/uncors/internal/config/...",
	"github.com/evg4b/uncors/internal/version/...",
}

// forbidden is every terminal library. The service must not reach for any of
// them: not the event loop, not the widgets, and not the styling. Rendering
// belongs to internal/render and internal/tui, which the service never imports.
var forbidden = []string{
	"charm.land/",
}

func dependenciesOf(t *testing.T, pattern string) []string {
	t.Helper()

	out, err := exec.Command("go", "list", "-deps", pattern).Output() //nolint:noctx // build-graph query
	require.NoError(t, err, "go list failed for %s", pattern)

	return strings.Split(strings.TrimSpace(string(out)), "\n")
}

// readGuardedSources exists for its side effect. The build graph is read by
// shelling out to go list, which Go's test cache cannot see, so a violating
// import could otherwise be masked by a cached pass. Opening the files the
// test guards puts them in the cache key.
func readGuardedSources(t *testing.T) {
	t.Helper()

	err := filepath.WalkDir("../../internal", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err //nolint:wrapcheck // walk error, propagated as-is
		}

		_, readErr := os.ReadFile(path) //nolint:gosec // fixed, repo-relative tree

		return readErr
	})
	require.NoError(t, err)
}

// T10: the service must not depend on any terminal library, directly or
// through anything it imports.
func TestServiceDoesNotDependOnTerminalLibraries(t *testing.T) {
	readGuardedSources(t)

	for _, pattern := range serviceSide {
		t.Run(pattern, func(t *testing.T) {
			deps := dependenciesOf(t, pattern)

			for _, dep := range deps {
				for _, banned := range forbidden {
					assert.NotContains(t, dep, banned,
						"%s must run without a terminal, but depends on %s", pattern, dep)
				}
			}
		})
	}
}

// The TUI is allowed to know about the service. The reverse is what breaks the
// architecture, and Go's import cycle rules do not catch it on their own.
func TestServiceDoesNotDependOnTheTUIPackage(t *testing.T) {
	readGuardedSources(t)

	for _, pattern := range serviceSide {
		t.Run(pattern, func(t *testing.T) {
			for _, dep := range dependenciesOf(t, pattern) {
				for _, presentation := range []string{
					"github.com/evg4b/uncors/internal/uncors_app",
					"github.com/evg4b/uncors/internal/tui",
					"github.com/evg4b/uncors/internal/render",
				} {
					assert.NotEqual(t, presentation, dep,
						"%s must not depend on the presentation layer", pattern)
				}
			}
		})
	}
}
