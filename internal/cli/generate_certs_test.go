package cli_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/cli"
	"github.com/evg4b/uncors/internal/di"
	"github.com/stretchr/testify/require"
)

func TestGenerateCerts(t *testing.T) {
	t.Run("returns error for unknown flag", func(t *testing.T) {
		err := cli.GenerateCerts(di.NewContainer(di.WithArgs([]string{"--unknown-flag"})))
		require.Error(t, err)
	})

	t.Run("generates CA certificate with valid args", func(t *testing.T) {
		// Point HOME to a temp dir so certs go there, not ~/.config/uncors.
		t.Setenv("HOME", t.TempDir())

		err := cli.GenerateCerts(di.NewContainer(di.WithArgs([]string{"--validity-days=7"})))
		require.NoError(t, err)
	})

	t.Run("returns nil for --help flag", func(t *testing.T) {
		err := cli.GenerateCerts(di.NewContainer(di.WithArgs([]string{"--help"})))
		require.NoError(t, err)
	})
}
