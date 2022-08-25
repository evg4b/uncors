package urlglob

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/evg4b/uncors/pkg/urlx"
)

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

func parsePattern(pattern string) (*url.URL, error) {
	parsedPttern, err := urlx.Parse(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	if len(parsedPttern.Fragment) > 0 || len(parsedPttern.RawQuery) > 0 || len(parsedPttern.Path) > 0 {
		return nil, ErrPatterntContinsData
	}

	return parsedPttern, nil
}
