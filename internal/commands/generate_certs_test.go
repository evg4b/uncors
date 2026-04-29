package commands_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/internal/commands"
	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	configDir  = ".config"
	caCertFile = "ca.crt"
	caKeyFile  = "ca.key"
)

func TestNewGenerateCertsCommand(t *testing.T) {
	t.Run("should create new command", func(t *testing.T) {
		fs := afero.NewOsFs()
		cmd := commands.NewGenerateCertsCommand(
			commands.WithFs(fs),
			commands.WithOutput(mocks.NoopOutput()),
		)
		assert.NotNil(t, cmd)
	})
}

func TestGenerateCertsCommand_DefineFlags(t *testing.T) {
	t.Run("should define validity-days flag", func(t *testing.T) {
		fs := afero.NewOsFs()
		cmd := commands.NewGenerateCertsCommand(
			commands.WithFs(fs),
			commands.WithOutput(mocks.NoopOutput()),
		)
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)

		cmd.DefineFlags(flags)

		flag := flags.Lookup("validity-days")
		assert.NotNil(t, flag)
		assert.Equal(t, "365", flag.DefValue)
	})

	t.Run("should define force flag", func(t *testing.T) {
		fs := afero.NewOsFs()
		cmd := commands.NewGenerateCertsCommand(
			commands.WithFs(fs),
			commands.WithOutput(mocks.NoopOutput()),
		)
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)

		cmd.DefineFlags(flags)

		flag := flags.Lookup("force")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})
}

func TestGenerateCertsCommand_Execute(t *testing.T) {
	t.Run("should generate CA certificate successfully", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()

		require.NoError(t, os.MkdirAll(fakeHome, 0o755))

		cmd := commands.NewGenerateCertsCommand(
			commands.WithFs(fs),
			commands.WithOutput(mocks.NoopOutput()),
		)
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
		cmd.DefineFlags(flags)

		err := cmd.Execute()
		require.NoError(t, err)

		caDir := filepath.Join(fakeHome, configDir, "uncors")
		assert.FileExists(t, filepath.Join(caDir, caCertFile))
		assert.FileExists(t, filepath.Join(caDir, caKeyFile))

		cert, key, err := infratls.LoadCA(fs,
			filepath.Join(caDir, caCertFile),
			filepath.Join(caDir, caKeyFile),
		)
		require.NoError(t, err)
		assert.NotNil(t, cert)
		assert.NotNil(t, key)
	})

	t.Run("should respect custom validity days", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()

		require.NoError(t, os.MkdirAll(fakeHome, 0o755))

		cmd := commands.NewGenerateCertsCommand(
			commands.WithFs(fs),
			commands.WithOutput(mocks.NoopOutput()),
		)
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
		cmd.DefineFlags(flags)

		err := flags.Set("validity-days", "730")
		require.NoError(t, err)

		err = cmd.Execute()
		require.NoError(t, err)

		caDir := filepath.Join(fakeHome, configDir, "uncors")
		cert, _, err := infratls.LoadCA(fs,
			filepath.Join(caDir, caCertFile),
			filepath.Join(caDir, caKeyFile),
		)
		require.NoError(t, err)

		duration := cert.NotAfter.Sub(cert.NotBefore)
		expectedDays := 730
		actualDays := int(duration.Hours() / 24)
		assert.InDelta(t, expectedDays, actualDays, 1.0)
	})

	t.Run("should fail when CA already exists without force flag", func(t *testing.T) {
		tmpDir := t.TempDir()

		fs := afero.NewOsFs()
		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		cmd1 := commands.NewGenerateCertsCommand(
			commands.WithFs(fs),
			commands.WithOutput(mocks.NoopOutput()),
		)
		flags1 := pflag.NewFlagSet("test", pflag.ContinueOnError)
		cmd1.DefineFlags(flags1)
		err := cmd1.Execute()
		require.NoError(t, err)

		cmd2 := commands.NewGenerateCertsCommand(
			commands.WithFs(fs),
			commands.WithOutput(mocks.NoopOutput()),
		)
		flags2 := pflag.NewFlagSet("test", pflag.ContinueOnError)
		cmd2.DefineFlags(flags2)
		err = cmd2.Execute()
		require.Error(t, err)
	})

	t.Run("should overwrite CA with force flag", func(t *testing.T) {
		tmpDir := t.TempDir()

		fs := afero.NewOsFs()
		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		cmd1 := commands.NewGenerateCertsCommand(
			commands.WithFs(fs),
			commands.WithOutput(mocks.NoopOutput()),
		)
		flags1 := pflag.NewFlagSet("test", pflag.ContinueOnError)
		cmd1.DefineFlags(flags1)
		err := cmd1.Execute()
		require.NoError(t, err)

		caDir := filepath.Join(fakeHome, configDir, "uncors")
		cert1, _, err := infratls.LoadCA(fs,
			filepath.Join(caDir, caCertFile),
			filepath.Join(caDir, caKeyFile),
		)
		require.NoError(t, err)

		cmd2 := commands.NewGenerateCertsCommand(
			commands.WithFs(fs),
			commands.WithOutput(mocks.NoopOutput()),
		)
		flags2 := pflag.NewFlagSet("test", pflag.ContinueOnError)
		cmd2.DefineFlags(flags2)
		err = flags2.Set("force", "true")
		require.NoError(t, err)

		err = cmd2.Execute()
		require.NoError(t, err)

		cert2, _, err := infratls.LoadCA(fs,
			filepath.Join(caDir, caCertFile),
			filepath.Join(caDir, caKeyFile),
		)
		require.NoError(t, err)
		assert.NotEqual(t, cert1.SerialNumber, cert2.SerialNumber)
	})

	t.Run("should create directory if not exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()
		cmd := commands.NewGenerateCertsCommand(
			commands.WithFs(fs),
			commands.WithOutput(mocks.NoopOutput()),
		)
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
		cmd.DefineFlags(flags)

		err := cmd.Execute()
		require.NoError(t, err)

		caDir := filepath.Join(fakeHome, configDir, "uncors")
		assert.DirExists(t, caDir)
	})

	for _, testCase := range []struct {
		name     string
		filename string
		content  []byte
		wantErr  error
	}{
		{"only cert file", caCertFile, []byte("cert"), commands.ErrCAAlreadyExists},
		{"only key file", caKeyFile, []byte("key"), commands.ErrCAKeyAlreadyExists},
	} {
		t.Run("should fail when "+testCase.name+" exists without force flag", func(t *testing.T) {
			tmpDir := t.TempDir()

			fakeHome := filepath.Join(tmpDir, "home")
			require.NoError(t, os.MkdirAll(fakeHome, 0o755))
			t.Setenv("HOME", fakeHome)

			caDir := filepath.Join(fakeHome, configDir, "uncors")
			require.NoError(t, os.MkdirAll(caDir, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(caDir, testCase.filename), testCase.content, 0o600))

			fs := afero.NewOsFs()
			cmd := commands.NewGenerateCertsCommand(
				commands.WithFs(fs),
				commands.WithOutput(mocks.NoopOutput()),
			)
			flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
			cmd.DefineFlags(flags)

			err := cmd.Execute()
			require.Error(t, err)
			assert.Equal(t, testCase.wantErr, err)
		})
	}
}
