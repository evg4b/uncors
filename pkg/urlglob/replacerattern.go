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

func NewReplacePatternString(rawURL string, options ...replacePatternOption) (ReplacePattern, error) {
	parsedPattern, err := parsePattern(rawURL)
	if err != nil {
		return ReplacePattern{}, fmt.Errorf("failed to craete glob from '%s': %w", rawURL, err)
	}

	return NewReplacePattern(parsedPattern, options...)
}

func NewReplacePattern(parsedPattern *url.URL, options ...replacePatternOption) (ReplacePattern, error) {
	rattern, wildCardCount, err := wildCardToReplacePattern(parsedPattern)
	if err != nil {
		return ReplacePattern{}, err
	}

	replacePattern := ReplacePattern{
		pattern:       rattern,
		wildCardCount: wildCardCount,
		scheme:        parsedPattern.Scheme,
		port:          parsedPattern.Port(),
	}

	PatchReplacePattern(&replacePattern, options...)

	return replacePattern, nil
}

func PatchReplacePattern(replacePattern *ReplacePattern, options ...replacePatternOption) {
	for _, option := range options {
		option(replacePattern)
	}
}
