package engine_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoreDoesNotImportTheUI(t *testing.T) {
	forbidden := []string{
		"github.com/evg4b/uncors/internal/tui",
		"github.com/evg4b/uncors/internal/uncors_app",
		"charm.land/",
		"github.com/charmbracelet/",
	}

	core := []string{
		"github.com/evg4b/uncors/internal/engine",
		"github.com/evg4b/uncors/internal/server",
		"github.com/evg4b/uncors/internal/certs",
		"github.com/evg4b/uncors/internal/events",
		"github.com/evg4b/uncors/internal/tracking",
		"github.com/evg4b/uncors/internal/telemetry",
		"github.com/evg4b/uncors/internal/handler/...",
	}

	args := append([]string{"list", "-deps", "-f", "{{.ImportPath}}"}, core...)

	output, err := exec.CommandContext(t.Context(), "go", args...).Output() //nolint:gosec // fixed arguments
	require.NoError(t, err)

	for dependency := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
		for _, prefix := range forbidden {
			assert.False(t, strings.HasPrefix(dependency, prefix), "the core depends on %s", dependency)
		}
	}
}

func TestPackagesDoNotImportInternal(t *testing.T) {
	args := []string{"list", "-deps", "-f", "{{.ImportPath}}", "github.com/evg4b/uncors/pkg/..."}

	output, err := exec.CommandContext(t.Context(), "go", args...).Output() //nolint:gosec // fixed arguments
	require.NoError(t, err)

	for dependency := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
		assert.False(t, strings.HasPrefix(dependency, "github.com/evg4b/uncors/internal/"),
			"pkg depends on %s", dependency)
	}
}
