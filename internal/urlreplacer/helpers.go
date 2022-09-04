package urlreplacer

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

func wildCardToRegexp(parsedPttern *url.URL) (*regexp.Regexp, int, error) {
	var result strings.Builder
	var count int

	result.WriteString(`^(?P<scheme>(http(s?):)?\/\/)?`)

	parts := strings.Split(parsedPttern.Hostname(), "*")
	for index, literal := range parts {
		if index > 0 {
			count++
			fmt.Fprintf(&result, "(?P<part%d>.+)", count)
		}

		_, err := result.WriteString(regexp.QuoteMeta(literal))
		if err != nil {
			return nil, 0, fmt.Errorf("filed to build url glob: %w", err)
		}
	}

	result.WriteString(`(?P<path>[\/?].*)?$`)

	regexp, err := regexp.Compile(result.String())
	if err != nil {
		return nil, 0, fmt.Errorf("filed to build url glob: %w", err)
	}

	return regexp, count, nil
}

func wildCardToReplacePattern(parsedPttern *url.URL) (string, int, error) {
	result := &strings.Builder{}
	var count int

	_, err := fmt.Fprint(result, "${scheme}")
	if err != nil {
		return "", count, fmt.Errorf("filed to build url glob: %w", err)
	}

	for i, literal := range strings.Split(parsedPttern.Hostname(), "*") {
		if i > 0 {
			count++
			fmt.Fprintf(result, "${part%d}", count)
		}

		_, err := fmt.Fprint(result, literal)
		if err != nil {
			return "", count, fmt.Errorf("filed to build url glob: %w", err)
		}
	}

	_, err = fmt.Fprint(result, "${path}")
	if err != nil {
		return "", count, fmt.Errorf("filed to build url glob: %w", err)
	}

	return result.String(), count, nil
}
