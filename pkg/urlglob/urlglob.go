package urlglob

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

type ReplacePattern struct {
	wildCardCount int
	pattern       string
	scheme        string
	port          string
}

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
)

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

func NewURLGlob(rawURL string) (*URLGlob, error) {
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

	return &URLGlob{
		regexp:         regexp,
		ReplacePattern: replacePattern,
		WildCardCount:  count,
		Scheme:         parsedPattern.Scheme,
		Port:           parsedPattern.Port(),
	}, nil
}

func parsePattern(pattern string) (*url.URL, error) {
	parsedPttern, err := url.Parse(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	if len(parsedPttern.Fragment) > 0 || len(parsedPttern.RawQuery) > 0 || len(parsedPttern.Path) > 0 {
		return nil, ErrPatterntContinsData
	}

	return parsedPttern, nil
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

	return regexp, count, err
}

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

func (glob *URLGlob) MatchString(rawURL string) (bool, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false, err
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
		return "", err
	}

	return glob.ReplaceAll(parsedURL, repl)
}

func (glob *URLGlob) ReplaceAll(parsedURL *url.URL, repl ReplacePattern) (string, error) {
	if !strings.EqualFold(glob.Scheme, parsedURL.Scheme) {
		return "", errors.New("fff") // TODO: add correct error
	}

	if len(repl.scheme) > 0 {
		parsedURL.Scheme = repl.scheme
	}

	if glob.WildCardCount < repl.wildCardCount {
		return "", errors.New("fff") // TODO: add correct error
	}

	hostname := glob.regexp.ReplaceAllString(parsedURL.Hostname(), repl.pattern)
	if len(repl.port) > 0 {
		parsedURL.Host = fmt.Sprintf("%s:%s", hostname, repl.port)
	} else {
		parsedURL.Host = hostname
	}

	return parsedURL.String(), nil
}
