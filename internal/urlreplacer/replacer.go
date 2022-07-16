package urlreplacer

import (
	"errors"
	"fmt"
	"net/url"
)

type urlData struct {
	hostname string
	host     string
	scheme   string
}

type Replacer struct {
	source urlData
	target urlData
}

var ErrSchemeNotMatched = errors.New("scheme in mapping and query not matched")

func (r *Replacer) ToSource(rawURL string) (string, error) {
	targetURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("filed transform url %s to source: %w", rawURL, err)
	}

	return r.URLToSource(targetURL)
}

func (r *Replacer) URLToSource(parsedURL *url.URL) (string, error) {
	expectedScheme := r.target.scheme
	if len(expectedScheme) > 0 && expectedScheme != parsedURL.Scheme {
		return "", fmt.Errorf(
			"failed to transform url from %s to %s: %w",
			expectedScheme,
			parsedURL.Scheme,
			ErrSchemeNotMatched,
		)
	}

	parsedURL.Host = r.source.host
	if len(r.source.scheme) > 0 {
		parsedURL.Scheme = r.source.scheme
	}

	return parsedURL.String(), nil
}

func (r *Replacer) ToTarget(rawURL string) (string, error) {
	targetURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("filed transform url %s to target: %w", rawURL, err)
	}

	return r.URLToTarget(targetURL)
}

func (r *Replacer) URLToTarget(parsedURL *url.URL) (string, error) {
	expectedScheme := r.source.scheme
	if len(expectedScheme) > 0 && expectedScheme != parsedURL.Scheme {
		return "", fmt.Errorf(
			"failed to transform url from %s to %s: %w",
			expectedScheme,
			parsedURL.Scheme,
			ErrSchemeNotMatched,
		)
	}

	parsedURL.Host = r.target.host
	if len(r.target.scheme) > 0 {
		parsedURL.Scheme = r.target.scheme
	}

	return parsedURL.String(), nil
}
