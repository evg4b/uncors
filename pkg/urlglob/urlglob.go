package urlglob

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/evg4b/uncors/pkg/urlx"
)

type URLGlob struct {
	WildCardCount int
	Port          string
	Scheme        string

	regexp *regexp.Regexp
}

var (
	ErrEmptyURL            = errors.New("url should not be empty")
	ErrPatternContainsData = errors.New("url pattern should not contain path, query or fragment")
	ErrSchemeMismatch      = errors.New("url scheme and mapping scheme is not equal")
	ErrTooManyWildcards    = errors.New("replace pattern contains too many wildcards")
)

func NewURLGlob(rawURL string) (*URLGlob, error) {
	if len(rawURL) == 0 {
		return nil, ErrEmptyURL
	}

	parsedPattern, err := parsePattern(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create glob from '%s': %w", rawURL, err)
	}

	regx, count, err := wildCardToRegexp(parsedPattern)
	if err != nil {
		return nil, err
	}

	glob := &URLGlob{
		regexp:        regx,
		WildCardCount: count,
		Scheme:        parsedPattern.Scheme,
		Port:          parsedPattern.Port(),
	}

	return glob, nil
}

func (glob *URLGlob) MatchString(rawURL string) (bool, error) {
	parsedURL, err := urlx.Parse(rawURL)
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

func (glob *URLGlob) ReplaceAllString(rawURL string, repl ReplacePattern) (*url.URL, error) {
	parsedURL, err := urlx.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("filed parse url for replacing: %w", err)
	}

	return glob.ReplaceAll(parsedURL, repl)
}

func (glob *URLGlob) ReplaceAll(parsedURL *url.URL, repl ReplacePattern) (*url.URL, error) {
	if len(glob.Scheme) > 0 && !strings.EqualFold(glob.Scheme, parsedURL.Scheme) {
		return nil, ErrSchemeMismatch
	}

	if repl.wildCardCount > glob.WildCardCount {
		return nil, ErrTooManyWildcards
	}

	if len(repl.scheme) > 0 {
		parsedURL.Scheme = repl.scheme
	}

	hostname := glob.regexp.ReplaceAllString(parsedURL.Hostname(), repl.pattern)
	if len(repl.port) > 0 {
		parsedURL.Host = fmt.Sprintf("%s:%s", hostname, repl.port)
	} else {
		parsedURL.Host = hostname
	}

	return parsedURL, nil
}
