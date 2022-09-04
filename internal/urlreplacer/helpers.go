package urlreplacer

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/evg4b/uncors/pkg/urlx"
)

func wildCardToRegexp(parsedPttern *url.URL) (*regexp.Regexp, int, error) {
	var result strings.Builder
	var count int

	for i, literal := range strings.Split(parsedPttern.Hostname(), "*") {
		if i > 0 {
			result.WriteString("(.+)")
		}

		_, err := result.WriteString(regexp.QuoteMeta(literal))
		if err != nil {
			return nil, 0, fmt.Errorf("filed to build url glob: %w", err)
		}

		count++
	}

	regexp, err := regexp.Compile(result.String())
	if err != nil {
		return nil, 0, fmt.Errorf("filed to build url glob: %w", err)
	}

	return regexp, count, nil
}

func wildCardToReplacePattern(parsedPttern *url.URL) (string, int, error) {
	var result strings.Builder
	var count int

	for i, literal := range strings.Split(parsedPttern.Hostname(), "*") {
		if i > 0 {
			fmt.Fprintf(&result, "$%d", i)
		}

		_, err := result.WriteString(literal)
		if err != nil {
			return "", count, fmt.Errorf("filed to build url glob: %w", err)
		}

		count++
	}

	return result.String(), count, nil
}

func isHost(host string) bool {
	parsed, err := urlx.Parse(host)
	if err != nil {
		return false
	}

	return strings.EqualFold(parsed.Host, host)
}
