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
	regexp      *regexp.Regexp
	pattern     string
	hooks       map[string]hook
	scheme      string
	subexpIndex map[string]int // precomputed name→index, built once at construction
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

	err := validateRawURL(source)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidSourceURL, err)
	}

	err = validateRawURL(target)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidTargetURL, err)
	}

	replacer := &Replacer{
		hooks: map[string]func(string) string{},
	}

	replacer.regexp, err = wildCardToRegexp(source)
	if err != nil {
		return nil, err
	}

	replacer.pattern = wildCardToReplacePattern(target)
	replacer.scheme = extractScheme(target)

	if len(replacer.scheme) > 0 {
		replacer.hooks["scheme"] = schemeHookFactory(replacer.scheme)
	}

	// Build name→index map once so Replace doesn't call SubexpIndex (O(n)) per group per call.
	replacer.subexpIndex = make(map[string]int, len(replacer.regexp.SubexpNames()))
	for i, name := range replacer.regexp.SubexpNames() {
		if name != "" {
			replacer.subexpIndex[name] = i
		}
	}

	return replacer, nil
}

func (r *Replacer) Replace(source string) (string, error) {
	matches := r.regexp.FindStringSubmatch(source)
	if len(matches) < 1 {
		return "", fmt.Errorf("url '%s' %w", source, ErrURLNotMatched)
	}

	replaced := strings.Clone(r.pattern)

	for name, idx := range r.subexpIndex {
		partValue := matches[idx]
		if hook, ok := r.hooks[name]; ok {
			partValue = hook(partValue)
		}

		replaced = strings.ReplaceAll(replaced, "${"+name+"}", partValue)
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
