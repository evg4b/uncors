package urlreplacer

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
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

// extractKeys returns the ordered placeholder key names from a raw URL pattern.
func extractKeys(raw string) []string {
	matches := placeholderRegexp.FindAllStringSubmatch(raw, -1)

	keys := make([]string, len(matches))
	for i, m := range matches {
		keys[i] = strings.ToLower(m[1])
	}

	return keys
}

// hasDuplicateKeys returns the first duplicate key name and true if one is found.
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

// rawHostPort extracts the "host:port" portion from a raw URL pattern string,
// stripping the scheme and any path/query/fragment.
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

// rawHost returns the host part of "host:port", validating that the port (if
// present) is not empty.
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

// wildCardToRegexp builds a regexp from a raw source URL pattern.
// {key} placeholders become named capture groups (?P<key>.+).
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

		key := host[match[0]+1 : match[1]-1] // strip { and }
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

// wildCardToReplacePattern builds a replacement pattern string from a raw
// target URL pattern. {key} placeholders become ${key} references.
func wildCardToReplacePattern(rawTarget string) string {
	result := &strings.Builder{}
	result.WriteString("${scheme}")

	hp := rawHostPort(rawTarget)
	replaced := placeholderRegexp.ReplaceAllStringFunc(hp, func(match string) string {
		key := match[1 : len(match)-1] // strip { and }

		return "${" + key + "}"
	})

	result.WriteString(replaced)
	result.WriteString("${path}")

	return result.String()
}

// validateRawURL checks that a raw URL pattern is valid: no path or query, no wildcard *.
// Only {key} placeholders are allowed, not * wildcards.
func validateRawURL(rawURL string) error {
	if len(rawURL) == 0 {
		return errEmptyURL
	}

	// Check for * wildcard usage - not allowed, use {key} placeholders instead
	if strings.Contains(rawURL, "*") {
		return errWildcardNotAllowed
	}

	// Replace {key} placeholders with a placeholder for validation
	// This allows standard url.Parse to validate the structure
	normalized := placeholderRegexp.ReplaceAllString(rawURL, "x")

	// Ensure URL has a scheme for proper parsing
	if !strings.Contains(normalized, "://") {
		if strings.HasPrefix(normalized, "//") {
			normalized = "http:" + normalized
		} else {
			normalized = "http://" + normalized
		}
	}

	// Validate with standard library
	parsed, err := url.Parse(normalized)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}

	// Check for path/query/fragment
	if len(parsed.Path) > 0 && parsed.Path != "/" {
		return errURLHasPath
	}

	if len(parsed.RawQuery) > 0 {
		return errURLHasQuery
	}

	return nil
}

// extractScheme returns the scheme (http or https) from a raw URL, or empty string.
func extractScheme(rawURL string) string {
	matches := schemeRegexp.FindStringSubmatch(rawURL)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}
