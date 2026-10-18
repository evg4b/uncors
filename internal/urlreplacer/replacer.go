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

var templatePartRegexp = regexp.MustCompile(`\$\{(\w+)\}`)

type templatePart struct {
	literal string
	group   int
	hook    hook
}

type Replacer struct {
	regexp   *regexp.Regexp
	template []templatePart
	scheme   string
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

	replacer := &Replacer{}

	replacer.regexp, err = wildCardToRegexp(source)
	if err != nil {
		return nil, err
	}

	replacer.scheme = extractScheme(target)

	hooks := map[string]hook{}
	if len(replacer.scheme) > 0 {
		hooks["scheme"] = schemeHookFactory(replacer.scheme)
	}

	replacer.template = compileTemplate(wildCardToReplacePattern(target), replacer.regexp, hooks)

	return replacer, nil
}

func compileTemplate(pattern string, source *regexp.Regexp, hooks map[string]hook) []templatePart {
	var parts []templatePart

	last := 0

	for _, match := range templatePartRegexp.FindAllStringSubmatchIndex(pattern, -1) {
		name := pattern[match[2]:match[3]]

		group := source.SubexpIndex(name)
		if group < 0 {
			continue
		}

		if match[0] > last {
			parts = append(parts, templatePart{literal: pattern[last:match[0]], group: -1})
		}

		parts = append(parts, templatePart{group: group, hook: hooks[name]})
		last = match[1]
	}

	if last < len(pattern) {
		parts = append(parts, templatePart{literal: pattern[last:], group: -1})
	}

	return parts
}

func (r *Replacer) Replace(source string) (string, error) {
	matches := r.regexp.FindStringSubmatch(source)
	if len(matches) < 1 {
		return "", fmt.Errorf("url '%s' %w", source, ErrURLNotMatched)
	}

	var builder strings.Builder

	builder.Grow(len(source))

	for _, part := range r.template {
		if part.group < 0 {
			builder.WriteString(part.literal)

			continue
		}

		value := matches[part.group]
		if part.hook != nil {
			value = part.hook(value)
		}

		builder.WriteString(value)
	}

	return builder.String(), nil
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
