package urlreplacer

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/evg4b/uncors/pkg/urlt"
)

var (
	placeholderRegexp     = regexp.MustCompile(`\{([a-zA-Z][a-zA-Z0-9_]*)\}`)
	schemeRegexp          = regexp.MustCompile(`^(https?):`)
	errEmptyPort          = errors.New("empty port")
	errEmptyURL           = errors.New("url is empty")
	errWildcardNotAllowed = errors.New("use {key} placeholders instead of * wildcard")
	errURLHasPath         = errors.New("url must not have a path")
	errURLHasQuery        = errors.New("url must not have query parameters")
)

func extractKeys(raw string) []string {
	matches := placeholderRegexp.FindAllStringSubmatch(raw, -1)

	keys := make([]string, len(matches))
	for i, m := range matches {
		keys[i] = strings.ToLower(m[1])
	}

	return keys
}

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

func rawHostPort(rawURL string) string {
	if i := strings.Index(rawURL, "://"); i >= 0 {
		rawURL = rawURL[i+3:]
	}

	rawURL = strings.TrimPrefix(rawURL, "//")

	if i := strings.IndexAny(rawURL, "/?#"); i >= 0 {
		rawURL = rawURL[:i]
	}

	return strings.ToLower(rawURL)
}

func rawHost(hostport string) (string, error) {
	if strings.HasPrefix(hostport, "[") {
		if i := strings.Index(hostport, "]"); i >= 0 {
			return hostport[:i+1], nil
		}
	}

	if i := strings.LastIndex(hostport, ":"); i >= 0 {
		if hostport[i+1:] == "" {
			return "", fmt.Errorf("failed to build url glob: port %q: %w", "//"+hostport, errEmptyPort)
		}

		return hostport[:i], nil
	}

	return hostport, nil
}

func wildCardToRegexp(rawSource string) (*regexp.Regexp, error) {
	var result strings.Builder

	result.WriteString(`^(?P<scheme>(http(s?):)?\/\/)?`)

	hp := rawHostPort(rawSource)

	host, err := rawHost(hp)
	if err != nil {
		return nil, err
	}

	lastIndex := 0
	for _, match := range placeholderRegexp.FindAllStringIndex(host, -1) {
		result.WriteString(regexp.QuoteMeta(host[lastIndex:match[0]]))

		key := host[match[0]+1 : match[1]-1]
		fmt.Fprintf(&result, "(?P<%s>.+)", key)

		lastIndex = match[1]
	}

	result.WriteString(regexp.QuoteMeta(host[lastIndex:]))
	result.WriteString(`(:\d+)?`)
	result.WriteString(`(?P<path>[\/?].*)?$`)

	compiled, err := regexp.Compile(result.String())
	if err != nil {
		return nil, fmt.Errorf("failed to build url glob: %w", err)
	}

	return compiled, nil
}

func wildCardToReplacePattern(rawTarget string) string {
	result := &strings.Builder{}
	result.WriteString("${scheme}")

	hp := rawHostPort(rawTarget)
	replaced := placeholderRegexp.ReplaceAllStringFunc(hp, func(match string) string {
		key := match[1 : len(match)-1]

		return "${" + key + "}"
	})

	result.WriteString(replaced)
	result.WriteString("${path}")

	return result.String()
}

func validateRawURL(rawURL string) error {
	if len(rawURL) == 0 {
		return errEmptyURL
	}

	if strings.Contains(rawURL, "*") {
		return errWildcardNotAllowed
	}

	normalized := placeholderRegexp.ReplaceAllString(rawURL, "x")

	if !strings.Contains(normalized, "://") && !strings.HasPrefix(normalized, "//") {
		normalized = "//" + normalized
	}

	parsed, err := urlt.Parse(normalized)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}

	if len(parsed.Path) > 0 && parsed.Path != "/" {
		return errURLHasPath
	}

	if len(parsed.RawQuery) > 0 {
		return errURLHasQuery
	}

	return nil
}

func extractScheme(rawURL string) string {
	matches := schemeRegexp.FindStringSubmatch(rawURL)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}
