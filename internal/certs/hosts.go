package certs

import (
	"net"
	"regexp"
	"strings"
)

var placeholderRegexp = regexp.MustCompile(`\{[a-zA-Z][a-zA-Z0-9_]*\}`)

const labelPattern = `[^.]+`

type hostMatcher struct {
	exact    map[string]struct{}
	patterns []*regexp.Regexp
}

func newHostMatcher(hosts []string) *hostMatcher {
	matcher := &hostMatcher{exact: make(map[string]struct{}, len(hosts))}

	for _, host := range hosts {
		host = normalizeHost(host)
		if !placeholderRegexp.MatchString(host) {
			matcher.exact[host] = struct{}{}

			continue
		}

		var pattern strings.Builder

		pattern.WriteString("^")

		lastIndex := 0
		for _, match := range placeholderRegexp.FindAllStringIndex(host, -1) {
			pattern.WriteString(regexp.QuoteMeta(host[lastIndex:match[0]]))
			pattern.WriteString(labelPattern)

			lastIndex = match[1]
		}

		pattern.WriteString(regexp.QuoteMeta(host[lastIndex:]))
		pattern.WriteString("$")

		matcher.patterns = append(matcher.patterns, regexp.MustCompile(pattern.String()))
	}

	return matcher
}

func (m *hostMatcher) match(host string) bool {
	if m == nil {
		return false
	}

	host = normalizeHost(host)
	if _, ok := m.exact[host]; ok {
		return true
	}

	for _, pattern := range m.patterns {
		if pattern.MatchString(host) {
			return true
		}
	}

	return false
}

func normalizeHost(host string) string {
	host = strings.TrimSuffix(strings.ToLower(host), ".")

	if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil {
		return ip.String()
	}

	return host
}
