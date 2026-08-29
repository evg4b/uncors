package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/internal/cli"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testVersion = "1.2.3"

func newEnv(t *testing.T, fs afero.Fs) (cli.Env, *bytes.Buffer) {
	t.Helper()

	buf := &bytes.Buffer{}

	return cli.Env{
		Fs:      fs,
		Stdout:  os.Stdout,
		Console: tui.NewCliOutput(buf),
		Version: testVersion,
	}, buf
}

func TestExecute(t *testing.T) {
	t.Run("prints the version and exits successfully", func(t *testing.T) {
		env, out := newEnv(t, afero.NewMemMapFs())

		code := cli.Execute(t.Context(), env, []string{"--version"})

		assert.Equal(t, 0, code)
		assert.Contains(t, out.String(), testVersion)
	})

	t.Run("a help request is not a failure", func(t *testing.T) {
		env, out := newEnv(t, afero.NewMemMapFs())

		code := cli.Execute(t.Context(), env, []string{"--help"})

		assert.Equal(t, 0, code)
		assert.Contains(t, out.String(), "generate-certs", "subcommands must be discoverable from help")
		assert.Contains(t, out.String(), "--interactive")
		assert.Contains(t, out.String(), "--version")
	})

	t.Run("a subcommand has its own help", func(t *testing.T) {
		env, out := newEnv(t, afero.NewMemMapFs())

		code := cli.Execute(t.Context(), env, []string{"generate-certs", "--help"})

		assert.Equal(t, 0, code)
		assert.Contains(t, out.String(), "--validity-days")
	})

	t.Run("reports an unknown flag", func(t *testing.T) {
		env, out := newEnv(t, afero.NewMemMapFs())

		code := cli.Execute(t.Context(), env, []string{"--no-such-flag"})

		assert.Equal(t, 1, code)
		assert.Contains(t, out.String(), "no-such-flag")
	})

	t.Run("reports an invalid configuration", func(t *testing.T) {
		env, out := newEnv(t, afero.NewMemMapFs())

		code := cli.Execute(t.Context(), env, []string{})

		assert.Equal(t, 1, code)
		assert.Contains(t, out.String(), "mappings")
	})

	t.Run("runs the generate-certs subcommand", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		env, _ := newEnv(t, afero.NewOsFs())

		code := cli.Execute(context.Background(), env, []string{"generate-certs"})

		require.Equal(t, 0, code)

		_, err := os.Stat(filepath.Join(home, ".config", "uncors", "ca.crt"))
		require.NoError(t, err)
	})
}
