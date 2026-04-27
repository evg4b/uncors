package urlreplacer

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrEmptySourceURL = errors.New("source url should not be empty")
	ErrEmptyTargetURL = errors.New("target url should not be empty")
)

var (
	ErrInvalidSourceURL   = errors.New("source host is invalid")
	ErrInvalidTargetURL   = errors.New("target host is invalid")
	ErrURLNotMatched      = errors.New("is not matched")
	ErrDuplicateSourceKey = errors.New("source url contains duplicate placeholder key")
	ErrURLHasPath         = errors.New("url must not have a path")
	ErrURLHasQuery        = errors.New("url must not have query parameters")
)

type hook = func(string) string

type Replacer struct {
	regexp  *regexp.Regexp
	pattern string
	hooks   map[string]hook
	scheme  string // target scheme (http or https), or empty
}

func NewReplacer(source, target string) (*Replacer, error) {
	if len(source) < 1 {
		return nil, ErrEmptySourceURL
	}

	if len(target) < 1 {
		return nil, ErrEmptyTargetURL
	}

	sourceKeys := extractKeys(source)
	if dup, isDup := hasDuplicateKeys(sourceKeys); isDup {
		return nil, fmt.Errorf("%w: {%s}", ErrDuplicateSourceKey, dup)
	}

	// Validate raw URLs before any processing
	if err := validateRawURL(source); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSourceURL, err)
	}

	if err := validateRawURL(target); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidTargetURL, err)
	}

	replacer := &Replacer{
		hooks: map[string]func(string) string{},
	}

	var err error
	replacer.regexp, err = wildCardToRegexp(source)
	if err != nil {
		return nil, err
	}

	replacer.pattern = wildCardToReplacePattern(target)

	// Extract and store target scheme
	replacer.scheme = extractScheme(target)
	if len(replacer.scheme) > 0 {
		replacer.hooks["scheme"] = schemeHookFactory(replacer.scheme)
	}

	return replacer, nil
}

func (r *Replacer) Replace(source string) (string, error) {
	matches := r.regexp.FindStringSubmatch(source)
	if len(matches) < 1 {
		return "", fmt.Errorf("url '%s' %w", source, ErrURLNotMatched)
	}

	replaced := strings.Clone(r.pattern)

	for _, subExpName := range r.regexp.SubexpNames() {
		if len(subExpName) > 0 {
			partPattern := fmt.Sprintf("${%s}", subExpName)
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
	replaced, err := r.Replace(source)
	if err == nil {
		return replaced
	}

	return source
}

func (r *Replacer) IsTargetSecure() bool {
	if len(r.scheme) > 0 {
		return isSecure(r.scheme)
	}

	return false
}

func (r *Replacer) IsMatched(source string) bool {
	return r.regexp.MatchString(source)
}

func isSecure(scheme string) bool {
	return strings.EqualFold(scheme, "https")
}
