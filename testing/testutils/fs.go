package testutils

import (
	"os"
	"testing"

	"github.com/spf13/afero"
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
