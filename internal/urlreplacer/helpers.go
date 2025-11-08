package urlreplacer

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/evg4b/uncors/internal/urlparser"
)

func wildCardToRegexp(parsedPattern *url.URL) (*regexp.Regexp, int, error) {
	var (
		result strings.Builder
		count  int
	)

	result.WriteString(`^(?P<scheme>(http(s?):)?\/\/)?`)

	host, _, err := urlparser.SplitHostPort(parsedPattern)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to build url glob: %w", err)
	}

	parts := strings.Split(host, "*")
	for index, literal := range parts {
		if index > 0 {
			count++
			fmt.Fprintf(&result, "(?P<part%d>.+)", count)
		}

		result.WriteString(regexp.QuoteMeta(literal))
	}

	result.WriteString(`(:\d+)?`)
	result.WriteString(`(?P<path>[\/?].*)?$`)

	compiledRegexp, err := regexp.Compile(result.String())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to build url glob: %w", err)
	}

	return compiledRegexp, count, nil
}

func wildCardToReplacePattern(parsedPattern *url.URL) (string, int) {
	result := &strings.Builder{}

	var count int

	result.WriteString("${scheme}")

	for i, literal := range strings.Split(parsedPattern.Host, "*") {
		if i > 0 {
			count++
			fmt.Fprintf(result, "${part%d}", count)
		}

		result.WriteString(literal)
	}

	result.WriteString("${path}")

	return result.String(), count
}
