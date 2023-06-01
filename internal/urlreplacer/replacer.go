package urlreplacer

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/evg4b/uncors/internal/sfmt"
	"github.com/evg4b/uncors/pkg/urlx"
)

var (
	ErrEmptySourceURL = errors.New("source url should not be empty")
	ErrEmptyTargetURL = errors.New("target url should not be empty")
)

var (
	ErrInvalidSourceURL = errors.New("source url is invalid")
	ErrURLNotMached     = errors.New("is not matched")
)

type hook = func(string) string

type Replacer struct {
	source  *url.URL
	target  *url.URL
	regexp  *regexp.Regexp
	pattern string
	hooks   map[string]hook
}

func NewReplacer(source, target string) (*Replacer, error) {
	if len(source) < 1 {
		return nil, ErrEmptySourceURL
	}

	if len(target) < 1 {
		return nil, ErrEmptyTargetURL
	}

	var err error
	replacer := Replacer{
		hooks: map[string]func(string) string{},
	}

	if replacer.source, err = urlx.Parse(source); err != nil {
		return nil, ErrInvalidSourceURL
	}

	if replacer.target, err = urlx.Parse(target); err != nil {
		return nil, ErrInvalidSourceURL
	}

	if replacer.regexp, _, err = wildCardToRegexp(replacer.source); err != nil {
		return nil, err
	}

	replacer.pattern, _ = wildCardToReplacePattern(replacer.target)

	if len(replacer.target.Scheme) > 0 {
		replacer.hooks["scheme"] = schemeHookFactory(replacer.target.Scheme)
	}

	return &replacer, nil
}

func (r *Replacer) Replace(source string) (string, error) {
	matches := r.regexp.FindStringSubmatch(source)
	if len(matches) < 1 {
		return "", fmt.Errorf("url '%s' %w", source, ErrURLNotMached)
	}

	replaced := strings.Clone(r.pattern)

	for _, subexpName := range r.regexp.SubexpNames() {
		if len(subexpName) > 0 {
			partPattern := sfmt.Sprintf("${%s}", subexpName)
			partIndex := r.regexp.SubexpIndex(subexpName)
			partValue := matches[partIndex]
			if hook, ok := r.hooks[subexpName]; ok {
				partValue = hook(partValue)
			}

			replaced = strings.ReplaceAll(replaced, partPattern, partValue)
		}
	}

	return replaced, nil
}

func (r *Replacer) IsSourceSecure() bool {
	if len(r.source.Scheme) > 0 {
		return isSecure(r.source.Scheme)
	}

	return false
}

func (r *Replacer) IsTargetSecure() bool {
	if len(r.target.Scheme) > 0 {
		return isSecure(r.target.Scheme)
	}

	return false
}

func (r *Replacer) IsMatched(source string) bool {
	return r.regexp.MatchString(source)
}

func isSecure(scheme string) bool {
	return strings.EqualFold(scheme, "https")
}
