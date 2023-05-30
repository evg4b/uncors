package testutils

import (
	"github.com/spf13/afero"
	"os"
	"path/filepath"
	"runtime"
	"testing"
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

func PrepareFsForTests(t *testing.T, folder string) afero.Fs {
	t.Helper()
	_, filename, _, _ := runtime.Caller(1)
	dirname := filepath.Join(filepath.Dir(filename), folder)

	return afero.NewReadOnlyFs(afero.NewBasePathFs(afero.NewOsFs(), dirname))
}
