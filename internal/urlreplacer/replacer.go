package urlreplacer

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/evg4b/uncors/pkg/urlglob"
)

type Replacer struct {
	rawSource string
	rawTarget string
	source    *urlglob.URLGlob
	target    *urlglob.URLGlob
}

var ErrSchemeNotMatched = errors.New("scheme in mapping and query not matched")

func (r *Replacer) ToSource(rawURL string) (string, error) {
	replcedURL, err := r.target.ReplaceAllString(rawURL, r.source.ReplacePattern)
	if err != nil {
		return "", fmt.Errorf("filed transform '%s' to source url:  %w", rawURL, err)
	}

	return replcedURL, nil
}

func (r *Replacer) URLToSource(parsedURL *url.URL) (string, error) {
	replcedURL, err := r.target.ReplaceAll(parsedURL, r.source.ReplacePattern)
	if err != nil {
		return "", fmt.Errorf("filed transform '%s' to source url:  %w", parsedURL.String(), err)
	}

	return replcedURL, nil
}

func (r *Replacer) ToTarget(rawURL string) (string, error) {
	replcedURL, err := r.source.ReplaceAllString(rawURL, r.target.ReplacePattern)
	if err != nil {
		return "", fmt.Errorf("filed transform '%s' to target url:  %w", rawURL, err)
	}

	return replcedURL, nil
}

func (r *Replacer) URLToTarget(parsedURL *url.URL) (string, error) {
	replcedURL, err := r.source.ReplaceAll(parsedURL, r.target.ReplacePattern)
	if err != nil {
		return "", fmt.Errorf("filed transform '%s' to target url:  %w", parsedURL.String(), err)
	}

	return replcedURL, nil
}
