package urlreplacer

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/evg4b/uncors/internal/sfmt"
	"github.com/evg4b/uncors/pkg/urlx"
)

func wildCardToRegexp(parsedPattern *url.URL) (*regexp.Regexp, int, error) {
	var result strings.Builder
	var count int

	result.WriteString(`^(?P<scheme>(http(s?):)?\/\/)?`)

	host, _, err := urlx.SplitHostPort(parsedPattern)
	if err != nil {
		return nil, 0, sfmt.Errorf("filed to build url glob: %w", err)
	}

	parts := strings.Split(host, "*")
	for index, literal := range parts {
		if index > 0 {
			count++
			sfmt.Fprintf(&result, "(?P<part%d>.+)", count)
		}

		if _, err := result.WriteString(regexp.QuoteMeta(literal)); err != nil {
			return nil, 0, sfmt.Errorf("filed to build url glob: %w", err)
		}
	}

	result.WriteString(`(:\d+)?`)
	result.WriteString(`(?P<path>[\/?].*)?$`)

	compiledRegexp, err := regexp.Compile(result.String())
	if err != nil {
		return nil, 0, sfmt.Errorf("filed to build url glob: %w", err)
	}

	return compiledRegexp, count, nil
}

func wildCardToReplacePattern(parsedPattern *url.URL) (string, int) {
	result := &strings.Builder{}
	var count int

	sfmt.Fprint(result, "${scheme}")

	for i, literal := range strings.Split(parsedPattern.Host, "*") {
		if i > 0 {
			count++
			sfmt.Fprintf(result, "${part%d}", count)
		}

		sfmt.Fprint(result, literal)
	}

	sfmt.Fprint(result, "${path}")

	return result.String(), count
}
