package cli_test

import (
	"io"
	"testing"

	"github.com/evg4b/uncors/internal/cli"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestGenerateCerts(t *testing.T) {
	t.Run("returns error for unknown flag", func(t *testing.T) {
		err := cli.GenerateCerts(cli.Env{Fs: afero.NewMemMapFs(), Stdout: io.Discard, Args: []string{"--unknown-flag"}})
		require.Error(t, err)
	})

	t.Run("generates CA certificate with valid args", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())

		err := cli.GenerateCerts(cli.Env{Fs: afero.NewMemMapFs(), Stdout: io.Discard, Args: []string{"--validity-days=7"}})
		require.NoError(t, err)
	})

	t.Run("returns nil for --help flag", func(t *testing.T) {
		err := cli.GenerateCerts(cli.Env{Fs: afero.NewMemMapFs(), Stdout: io.Discard, Args: []string{"--help"}})
		require.NoError(t, err)
	})
}
