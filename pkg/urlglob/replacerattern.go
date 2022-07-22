package urlglob

import (
	"fmt"
	"net/url"
)

type ReplacePattern struct {
	wildCardCount int
	pattern       string
	scheme        string
	port          string
}

func NewReplacePatternString(rawURL string) (ReplacePattern, error) {
	parsedPattern, err := parsePattern(rawURL)
	if err != nil {
		return ReplacePattern{}, fmt.Errorf("failed to craete glob from '%s': %w", rawURL, err)
	}

	return NewReplacePattern(parsedPattern)
}

func NewReplacePattern(parsedPattern *url.URL) (ReplacePattern, error) {
	replacePattern, wildCardCount, err := wildCardToReplacePattern(parsedPattern)
	if err != nil {
		return ReplacePattern{}, err
	}

	return ReplacePattern{
		pattern:       replacePattern,
		wildCardCount: wildCardCount,
		scheme:        parsedPattern.Scheme,
		port:          parsedPattern.Port(),
	}, nil
}
