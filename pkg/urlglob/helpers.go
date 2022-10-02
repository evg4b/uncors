package urlglob

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/evg4b/uncors/pkg/urlx"
)

func wildCardToReplacePattern(parsedPattern *url.URL) (string, int, error) {
	var result strings.Builder
	var count int

	for i, literal := range strings.Split(parsedPattern.Hostname(), "*") {
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

func wildCardToRegexp(parsedPattern *url.URL) (*regexp.Regexp, int, error) {
	var result strings.Builder
	var count int

	for i, literal := range strings.Split(parsedPattern.Hostname(), "*") {
		if i > 0 {
			result.WriteString("(.+)")
		}

		_, err := result.WriteString(regexp.QuoteMeta(literal))
		if err != nil {
			return nil, 0, fmt.Errorf("filed to build url glob: %w", err)
		}

		count++
	}

	regx, err := regexp.Compile(result.String())
	if err != nil {
		return nil, 0, fmt.Errorf("filed to build url glob: %w", err)
	}

	return regx, count, nil
}

func parsePattern(pattern string) (*url.URL, error) {
	parsedPattern, err := urlx.Parse(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	if len(parsedPattern.Fragment) > 0 || len(parsedPattern.RawQuery) > 0 || len(parsedPattern.Path) > 0 {
		return nil, ErrPatternContainsData
	}

	return parsedPattern, nil
}
