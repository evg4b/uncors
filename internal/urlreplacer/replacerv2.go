package urlreplacer

import (
	"errors"
	"net/url"
	"regexp"

	"github.com/evg4b/uncors/pkg/urlx"
)

var ErrEmptySourceURL = errors.New("source url should not be empty")
var ErrEmptyTargetURL = errors.New("target url should not be empty")

var ErrInvalidSourceURL = errors.New("source url is invalid")
var ErrInvalidTargetURL = errors.New("target url is invalid")

type ReplacerV2 struct {
	source  *url.URL
	target  *url.URL
	regexp  *regexp.Regexp
	pattern string
}

func NewReplacerV2(source, target string) (*ReplacerV2, error) {
	if len(source) < 1 {
		return nil, ErrEmptySourceURL
	}

	if len(target) < 1 {
		return nil, ErrEmptyTargetURL
	}

	var err error
	replacer := ReplacerV2{}
	if replacer.source, err = url.Parse(source); err != nil {
		return nil, ErrInvalidSourceURL
	}

	if replacer.target, err = url.Parse(target); err != nil {
		return nil, ErrInvalidSourceURL
	}

	replacer.regexp, _, _ = wildCardToRegexp(replacer.source)
	replacer.pattern, _, _ = wildCardToReplacePattern(replacer.target)

	return &replacer, nil
}

func (r *ReplacerV2) Replace(source string) (string, error) {
	parsed, err := urlx.Parse(source)
	if err != nil {
		return "", err
	}

	transformed, err := r.ReplaceURL(parsed)
	if err != nil {
		return "", err
	}

	if isHost(source) {
		return transformed.Host, nil
	}

	return transformed.String(), nil
}

func (r *ReplacerV2) ReplaceURL(source *url.URL) (*url.URL, error) {
	target := *source
	hostname := r.regexp.ReplaceAllString(source.Host, r.pattern)
	target.Host = hostname
	if len(r.target.Scheme) > 0 {
		target.Scheme = r.target.Scheme
	}

	return &target, nil
}
