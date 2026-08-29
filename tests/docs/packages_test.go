package docs_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// packageReference matches an `internal/...` path written in backticks.
var packageReference = regexp.MustCompile("`(internal/[a-z0-9_/]+)`")

// The architecture documents used to name packages that had been renamed or
// removed. Naming a package that does not exist is worse than saying nothing:
// it is specific, so a reader trusts it.
func TestDocumentedPackagesExist(t *testing.T) {
	for _, doc := range []string{"../../ARCHITECTURE.md", "../../CLAUDE.md", "../../CONTRIBUTING.md"} {
		t.Run(filepath.Base(doc), func(t *testing.T) {
			content, err := os.ReadFile(doc)
			require.NoError(t, err)

			for _, match := range packageReference.FindAllStringSubmatch(string(content), -1) {
				pkg := match[1]

				t.Run(pkg, func(t *testing.T) {
					// The path comes from a regexp over our own documentation.
					//nolint:gosec // G703
					info, err := os.Stat(filepath.Join("../..", pkg))

					require.NoError(t, err, "%s names %s, which does not exist", doc, pkg)
					assert.True(t, info.IsDir())
				})
			}
		})
	}
}
