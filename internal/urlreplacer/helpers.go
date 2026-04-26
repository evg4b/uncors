package urlreplacer

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/evg4b/uncors/internal/urlparser"
)

// placeholderRegexp matches named URL placeholders like {client} or {region}.
var placeholderRegexp = regexp.MustCompile(`\{([a-zA-Z][a-zA-Z0-9_]*)\}`)

// extractKeys returns the ordered list of placeholder key names from a raw URL pattern.
// For example, "http://{client}.{region}.com" returns ["client", "region"].
func extractKeys(raw string) []string {
	matches := placeholderRegexp.FindAllStringSubmatch(raw, -1)
	keys := make([]string, len(matches))
	for i, m := range matches {
		keys[i] = m[1]
	}

	return keys
}

// hasDuplicateKeys checks whether keys contains any duplicate.
// Returns the duplicate key name and true if a duplicate is found.
func hasDuplicateKeys(keys []string) (string, bool) {
	seen := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		if _, exists := seen[k]; exists {
			return k, true
		}

		seen[k] = struct{}{}
	}

	return "", false
}

func wildCardToRegexp(parsedPattern *url.URL, keys []string) (*regexp.Regexp, error) {
	var (
		result   strings.Builder
		keyIndex int
	)

	result.WriteString(`^(?P<scheme>(http(s?):)?\/\/)?`)

	host, _, err := urlparser.SplitHostPort(parsedPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to build url glob: %w", err)
	}

	parts := strings.Split(host, "*")
	for index, literal := range parts {
		if index > 0 {
			name := fmt.Sprintf("part%d", keyIndex+1)
			if keyIndex < len(keys) {
				name = keys[keyIndex]
			}
			keyIndex++
			fmt.Fprintf(&result, "(?P<%s>.+)", name)
		}

		result.WriteString(regexp.QuoteMeta(literal))
	}

	result.WriteString(`(:\d+)?`)
	result.WriteString(`(?P<path>[\/?].*)?$`)

	compiledRegexp, err := regexp.Compile(result.String())
	if err != nil {
		return nil, fmt.Errorf("failed to build url glob: %w", err)
	}

	return compiledRegexp, nil
}

func wildCardToReplacePattern(parsedPattern *url.URL, keys []string) string {
	result := &strings.Builder{}
	result.WriteString("${scheme}")

	keyIndex := 0
	for i, literal := range strings.Split(parsedPattern.Host, "*") {
		if i > 0 {
			name := fmt.Sprintf("part%d", keyIndex+1)
			if keyIndex < len(keys) {
				name = keys[keyIndex]
			}
			keyIndex++
			fmt.Fprintf(result, "${%s}", name)
		}

		result.WriteString(literal)
	}

	result.WriteString("${path}")

	return result.String()
}
