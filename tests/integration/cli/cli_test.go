package cli_test

import (
	"fmt"
	"regexp"
	"runtime"
	"testing"

	"github.com/evg4b/uncors/testing/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	expected := regexp.MustCompile(fmt.Sprintf(
		`^version: %s\ngo:      %s\ncommit:  ([0-9a-f]{40}(-dirty)?|unknown)\n$`,
		regexp.QuoteMeta(integration.UncorsTestVersion),
		regexp.QuoteMeta(runtime.Version()),
	))

	t.Run("short", func(t *testing.T) {
		cmd := integration.UncorsCommand(t, []string{"-v"})
		bytes, err := cmd.CombinedOutput()
		require.NoError(t, err)

		assert.Regexp(t, expected, string(bytes))
	})

	t.Run("full", func(t *testing.T) {
		cmd := integration.UncorsCommand(t, []string{"--version"})
		bytes, err := cmd.CombinedOutput()
		require.NoError(t, err)

		assert.Regexp(t, expected, string(bytes))
	})
}
