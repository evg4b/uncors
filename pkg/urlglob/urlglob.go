package urlglob

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

type URLGlob struct {
	regexp         *regexp.Regexp
	ReplacePattern ReplacePattern
	WildCardCount  int
	Port           string
	Scheme         string
}

var (
	ErrEmptyURL            = errors.New("url should not be empty")
	ErrPatterntContinsData = errors.New("url pattern should not contain path, query or fragment")
	ErrSchemeMismatch      = errors.New("url scheme and mapping scheme is not equal")
	ErrTooManyWildcards    = errors.New("replcace pattern contains too many wildcards")
)

func NewURLGlob(rawURL string, options ...urlGloboption) (*URLGlob, error) {
	if len(rawURL) == 0 {
		return nil, ErrEmptyURL
	}
	parsedPattern, err := parsePattern(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to craete glob from '%s': %w", rawURL, err)
	}

	regexp, count, err := wildCardToRegexp(parsedPattern)
	if err != nil {
		return nil, err
	}

	replacePattern, err := NewReplacePattern(parsedPattern)
	if err != nil {
		return nil, err
	}

	glob := &URLGlob{
		regexp:         regexp,
		ReplacePattern: replacePattern,
		WildCardCount:  count,
		Scheme:         parsedPattern.Scheme,
		Port:           parsedPattern.Port(),
	}

	for _, option := range options {
		option(glob)
	}

	return glob, nil
}

func (glob *URLGlob) MatchString(rawURL string) (bool, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false, fmt.Errorf("filed parse url for matching: %w", err)
	}

	return glob.Match(parsedURL), nil
}

func (glob *URLGlob) Match(parsedURL *url.URL) bool {
	if len(glob.Scheme) > 0 && !strings.EqualFold(glob.Scheme, parsedURL.Scheme) {
		return false
	}

	return glob.regexp.MatchString(parsedURL.Host)
}

func (glob *URLGlob) ReplaceAllString(rawURL string, repl ReplacePattern) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("filed parse url for replacing: %w", err)
	}

	return glob.ReplaceAll(parsedURL, repl)
}

func (glob *URLGlob) ReplaceAll(parsedURL *url.URL, repl ReplacePattern) (string, error) {
	if !strings.EqualFold(glob.Scheme, parsedURL.Scheme) {
		return "", ErrSchemeMismatch
	}

	if len(repl.scheme) > 0 {
		parsedURL.Scheme = repl.scheme
	}

	if glob.WildCardCount < repl.wildCardCount {
		return "", ErrTooManyWildcards
	}

	hostname := glob.regexp.ReplaceAllString(parsedURL.Hostname(), repl.pattern)
	if len(repl.port) > 0 {
		parsedURL.Host = fmt.Sprintf("%s:%s", hostname, repl.port)
	} else {
		parsedURL.Host = hostname
	}

	return parsedURL.String(), nil
}
