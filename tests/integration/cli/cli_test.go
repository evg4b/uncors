package cli_test

import (
	"fmt"
	"testing"

	"github.com/evg4b/uncors/testing/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	t.Run("short", func(t *testing.T) {
		cmd := integration.UncorsCommand(t, []string{"-v"})
		bytes, err := cmd.CombinedOutput()
		require.NoError(t, err)

		assert.Equal(t, fmt.Sprintf("%s\n", integration.UncorsTestVrsion), string(bytes))
	})

	t.Run("full", func(t *testing.T) {
		cmd := integration.UncorsCommand(t, []string{"--version"})
		bytes, err := cmd.CombinedOutput()
		require.NoError(t, err)

		assert.Equal(t, fmt.Sprintf("%s\n", integration.UncorsTestVrsion), string(bytes))
	})
}
