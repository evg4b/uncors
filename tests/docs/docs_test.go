// Package docs_test parses every configuration example in the documentation, so
// that a documented config which cannot be loaded fails the build.
//
// This is the check that would have caught a documented duration spelling that
// did not parse and a documented flag that did not exist.
package docs_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// yamlBlock matches a fenced ```yaml block.
var yamlBlock = regexp.MustCompile("(?s)```yaml\n(.*?)```")

// configLike recognises the blocks that are meant to be loadable uncors
// configuration. Skipped are unrelated YAML (a CI workflow), skeletons that use
// "..." as a placeholder for omitted content, and the Migration Guide, whose
// examples deliberately show formats that no longer parse.
func configLike(path, block string) bool {
	if filepath.Base(path) == "Migration-Guide.md" {
		return false
	}

	if strings.Contains(block, "...") {
		return false
	}

	// A block without mappings is a fragment illustrating one section, not a
	// configuration a user could run.
	return strings.Contains(block, "mappings:")
}

func TestDocumentedConfigurationsLoad(t *testing.T) {
	files, err := filepath.Glob("../../docs/*.md")
	require.NoError(t, err)

	files = append(files, "../../README.md")

	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			content, err := os.ReadFile(path)
			require.NoError(t, err)

			for index, match := range yamlBlock.FindAllStringSubmatch(string(content), -1) {
				block := match[1]
				if !configLike(path, block) {
					continue
				}

				t.Run("example "+strconv.Itoa(index), func(t *testing.T) {
					loadExample(t, block)
				})
			}
		})
	}
}

// loadExample runs one documented example through the real loader. Examples
// refer to files and directories that do not exist on this machine, so they are
// loaded against a filesystem that reports every path as present.
func loadExample(t *testing.T, block string) {
	t.Helper()

	fs := everythingExistsFs{Fs: afero.NewMemMapFs()}

	require.NoError(t, afero.WriteFile(fs, "/docs-example.yaml", []byte(block), 0o600))

	flags, err := config.ParseFlags([]string{"--config", "/docs-example.yaml"})
	require.NoError(t, err)

	_, err = config.LoadConfiguration(fs, flags)
	require.NoError(t, err, "this configuration is documented but does not load:\n%s", block)
}

// everythingExistsFs reports any path as an existing file, so that examples
// referring to ./dist or ~/mocks validate on a machine that has neither.
type everythingExistsFs struct {
	afero.Fs
}

func (fs everythingExistsFs) Stat(name string) (os.FileInfo, error) {
	info, err := fs.Fs.Stat(name)
	if err == nil {
		return info, nil
	}

	return stubFileInfo{name: filepath.Base(name)}, nil
}

func (fs everythingExistsFs) Open(name string) (afero.File, error) {
	file, err := fs.Fs.Open(name)
	if err == nil {
		return file, nil
	}

	return fs.Create(name)
}

type stubFileInfo struct {
	os.FileInfo

	name string
}

func (i stubFileInfo) Name() string { return i.name }
func (i stubFileInfo) Size() int64  { return 0 }

// IsDir guesses from the name: documented static roots are "./dist", documented
// mock bodies are "./body.json".
func (i stubFileInfo) IsDir() bool {
	return !strings.Contains(i.name, ".")
}
