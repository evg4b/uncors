package cli_test

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// documentedFlag matches a `| --name | -s | description |` row of the CLI table.
var documentedFlag = regexp.MustCompile(`^\|\s*` + "`" + `(--[a-z-]+)` + "`")

// The CLI reference in the docs is hand-written, which is how it came to
// document a --debug flag that did not exist while omitting --interactive. This
// pins the table to the actual flag set.
func TestDocumentedFlagsMatchTheFlagSet(t *testing.T) {
	const docPath = "../../docs/Configuration.md"

	content, err := os.ReadFile(docPath)
	require.NoError(t, err)

	documented := map[string]bool{}

	for line := range strings.SplitSeq(string(content), "\n") {
		match := documentedFlag.FindStringSubmatch(strings.TrimSpace(line))
		if match != nil {
			documented[match[1]] = true
		}
	}

	set := pflag.NewFlagSet("uncors", pflag.ContinueOnError)
	config.DefineFlags(set)

	// --version is defined by the command layer for every command.
	actual := map[string]bool{"--version": true}

	set.VisitAll(func(flag *pflag.Flag) {
		actual["--"+flag.Name] = true
	})

	assert.Equal(t, sortedKeys(actual), sortedKeys(documented),
		"docs/Configuration.md must document exactly the flags uncors accepts")
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}
