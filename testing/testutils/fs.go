package testutils

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// FsFromMap creates afero.Fs in memory from map.
// Where key is a filename and value is file context.
func FsFromMap(t *testing.T, files map[string]string) afero.Fs {
	t.Helper()

	fs := afero.NewMemMapFs()
	for path, content := range files {
		err := afero.WriteFile(fs, path, []byte(content), os.ModePerm)
		if err != nil {
			t.Error(err)
		}
	}

	return fs
}

func CurrentDir(t *testing.T) string {
	t.Helper()
	_, callerFile, _, ok := runtime.Caller(1)
	require.True(t, ok, "Failed to get caller information")

	return filepath.Dir(callerFile)
}
