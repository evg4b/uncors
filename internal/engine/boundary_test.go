package engine_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The proxy core must run without any user interface, so it may not depend on
// the UI packages or the terminal libraries, even indirectly.
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
		"github.com/evg4b/uncors/internal/events",
		"github.com/evg4b/uncors/internal/tracking",
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
