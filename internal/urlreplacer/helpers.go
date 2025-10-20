package urlreplacer

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/urlparser"
)

func wildCardToRegexp(parsedPattern *url.URL) (*regexp.Regexp, int, error) {
	var result strings.Builder
	var count int

	result.WriteString(`^(?P<scheme>(http(s?):)?\/\/)?`)

	host, _, err := urlparser.SplitHostPort(parsedPattern)
	if err != nil {
		return nil, 0, fmt.Errorf("filed to build url glob: %w", err)
	}

	parts := strings.Split(host, "*")
	for index, literal := range parts {
		if index > 0 {
			count++
			helpers.FPrintf(&result, "(?P<part%d>.+)", count)
		}

		if _, err := result.WriteString(regexp.QuoteMeta(literal)); err != nil {
			return nil, 0, fmt.Errorf("filed to build url glob: %w", err)
		}
	}

	result.WriteString(`(:\d+)?`)
	result.WriteString(`(?P<path>[\/?].*)?$`)

	compiledRegexp, err := regexp.Compile(result.String())
	if err != nil {
		return nil, 0, fmt.Errorf("filed to build url glob: %w", err)
	}

	return compiledRegexp, count, nil
}

func wildCardToReplacePattern(parsedPattern *url.URL) (string, int) {
	result := &strings.Builder{}
	var count int

	helpers.FPrint(result, "${scheme}")

	for i, literal := range strings.Split(parsedPattern.Host, "*") {
		if i > 0 {
			count++
			helpers.FPrintf(result, "${part%d}", count)
		}

		helpers.FPrint(result, literal)
	}

	helpers.FPrint(result, "${path}")

	return result.String(), count
}
