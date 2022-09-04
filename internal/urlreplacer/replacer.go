package urlreplacer

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/evg4b/uncors/pkg/urlglob"
)

type Replacer struct {
	rawSource            string
	rawTarget            string
	source               *urlglob.URLGlob
	sourceReplacePattern urlglob.ReplacePattern
	sourceHasTLS         bool
	target               *urlglob.URLGlob
	targetReplacePattern urlglob.ReplacePattern
	targetHasTLS         bool
}

var ErrSchemeNotMatched = errors.New("scheme in mapping and query not matched")

func (r *Replacer) StringToSource(rawURL string) (string, error) {
	replcedURL, err := r.target.ReplaceAllString(rawURL, r.sourceReplacePattern)
	if err != nil {
		return "", fmt.Errorf("filed transform '%s' to source url:  %w", rawURL, err)
	}

	return replcedURL.String(), nil
}

func (r *Replacer) URLToSource(parsedURL *url.URL) (*url.URL, error) {
	replcedURL, err := r.target.ReplaceAll(parsedURL, r.sourceReplacePattern)
	if err != nil {
		return nil, fmt.Errorf("filed transform '%s' to source url:  %w", parsedURL.String(), err)
	}

	return replcedURL, nil
}

func (r *Replacer) ToTarget(rawURL string) (string, error) {
	replcedURL, err := r.source.ReplaceAllString(rawURL, r.targetReplacePattern)
	if err != nil {
		return "", fmt.Errorf("filed transform '%s' to target url:  %w", rawURL, err)
	}

	return replcedURL.String(), nil
}

func (r *Replacer) URLToTarget(parsedURL *url.URL) (*url.URL, error) {
	replcedURL, err := r.source.ReplaceAll(parsedURL, r.targetReplacePattern)
	if err != nil {
		return nil, fmt.Errorf("filed transform '%s' to target url:  %w", parsedURL.String(), err)
	}

	return replcedURL, nil
}

func (r *Replacer) IsSourceSecure() bool {
	return r.sourceHasTLS
}

func (r *Replacer) IsTargetSecure() bool {
	return r.targetHasTLS
}
