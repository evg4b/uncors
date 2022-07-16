package urlreplacer

import (
	"fmt"
	"net/url"
)

type urlData struct {
	hostname string
	host     string
	scheme   string
}

type replacer struct {
	source urlData
	target urlData
}

func (r *replacer) ToSource(rawUrl string) (string, error) {
	targetUrl, err := url.Parse(rawUrl)
	if err != nil {
		return "", fmt.Errorf("filed transform url %s to source: %v", rawUrl, err)
	}

	return r.UrlToSource(targetUrl)
}

func (r *replacer) UrlToSource(parsedUrl *url.URL) (string, error) {
	expectedScheme := r.target.scheme
	if len(expectedScheme) > 0 && expectedScheme != parsedUrl.Scheme {
		return "", fmt.Errorf("target url scheme in mapping (%s) and in query (%s) are not equal", expectedScheme, parsedUrl.Scheme)
	}

	parsedUrl.Host = r.source.host
	if len(r.source.scheme) > 0 {
		parsedUrl.Scheme = r.source.scheme
	}

	return parsedUrl.String(), nil
}

func (r *replacer) ToTarget(rawUrl string) (string, error) {
	targetUrl, err := url.Parse(rawUrl)
	if err != nil {
		return "", fmt.Errorf("filed transform url %s to target: %v", rawUrl, err)
	}

	return r.UrlToTarget(targetUrl)
}

func (r *replacer) UrlToTarget(parsedUrl *url.URL) (string, error) {
	expectedScheme := r.source.scheme
	if len(expectedScheme) > 0 && expectedScheme != parsedUrl.Scheme {
		return "", fmt.Errorf("target url scheme in mapping (%s) and in query (%s) are not equal", expectedScheme, parsedUrl.Scheme)
	}

	parsedUrl.Host = r.target.host
	if len(r.target.scheme) > 0 {
		parsedUrl.Scheme = r.target.scheme
	}

	return parsedUrl.String(), nil
}
