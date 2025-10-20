package urlreplacer

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/urlparser"
)

var (
	ErrEmptySourceURL = errors.New("source url should not be empty")
	ErrEmptyTargetURL = errors.New("target url should not be empty")
)

var (
	ErrInvalidSourceURL = errors.New("source host is invalid")
	ErrInvalidTargetURL = errors.New("target host is invalid")
	ErrURLNotMatched    = errors.New("is not matched")
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
	replacer := &Replacer{
		hooks: map[string]func(string) string{},
	}

	if replacer.source, err = urlparser.Parse(source); err != nil {
		return nil, ErrInvalidSourceURL
	}

	if replacer.target, err = urlparser.Parse(target); err != nil {
		return nil, ErrInvalidTargetURL
	}

	if replacer.regexp, _, err = wildCardToRegexp(replacer.source); err != nil {
		return nil, err
	}

	replacer.pattern, _ = wildCardToReplacePattern(replacer.target)

	if len(replacer.target.Scheme) > 0 {
		replacer.hooks["scheme"] = schemeHookFactory(replacer.target.Scheme)
	}

	return replacer, validateReplacer(replacer)
}

func validateReplacer(replacer *Replacer) error {
	if len(replacer.source.Path) > 0 || len(replacer.source.RawQuery) > 0 {
		return ErrInvalidSourceURL
	}

	if len(replacer.target.Path) > 0 || len(replacer.target.RawQuery) > 0 {
		return ErrInvalidTargetURL
	}

	return nil
}

func (r *Replacer) Replace(source string) (string, error) {
	matches := r.regexp.FindStringSubmatch(source)
	if len(matches) < 1 {
		return "", fmt.Errorf("url '%s' %w", source, ErrURLNotMatched)
	}

	replaced := strings.Clone(r.pattern)

	for _, subExpName := range r.regexp.SubexpNames() {
		if len(subExpName) > 0 {
			partPattern := helpers.Sprintf("${%s}", subExpName)
			partIndex := r.regexp.SubexpIndex(subExpName)
			partValue := matches[partIndex]
			if hook, ok := r.hooks[subExpName]; ok {
				partValue = hook(partValue)
			}

			replaced = strings.ReplaceAll(replaced, partPattern, partValue)
		}
	}

	return replaced, nil
}

func (r *Replacer) ReplaceSoft(source string) string {
	if replaced, err := r.Replace(source); err == nil {
		return replaced
	}

	return source
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
