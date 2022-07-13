package urlreplacer

import (
	"fmt"
	"net/url"
)

type replceData struct {
	hostname string
	host     string
	scheme   string
}

type replacer struct {
	source replceData
	target replceData
}

func (r *replacer) ToSource(rawTargetUrl string) (string, error) {
	targetUrl, err := url.Parse(rawTargetUrl)
	if err != nil {
		return "", fmt.Errorf("filed transform url %s to source: %v", rawTargetUrl, err)
	}

	return r.UrlToSource(targetUrl)
}

func (r *replacer) UrlToSource(targetUrl *url.URL) (string, error) {
	expectedScheme := r.target.scheme
	if len(expectedScheme) > 0 && expectedScheme != targetUrl.Scheme {
		return "", fmt.Errorf("target url scheme in mapping (%s) and in query (%s) are not equal", expectedScheme, targetUrl.Scheme)
	}

	targetUrl.Host = r.source.host
	if len(r.source.scheme) > 0 {
		targetUrl.Scheme = r.source.scheme
	}

	return targetUrl.String(), nil
}

func (r *replacer) ToTarget(rawTargetUrl string) (string, error) {
	targetUrl, err := url.Parse(rawTargetUrl)
	if err != nil {
		return "", fmt.Errorf("filed transform url %s to target: %v", rawTargetUrl, err)
	}

	return r.UrlToTarget(targetUrl)
}

func (r *replacer) UrlToTarget(targetUrl *url.URL) (string, error) {
	expectedScheme := r.source.scheme
	if len(expectedScheme) > 0 && expectedScheme != targetUrl.Scheme {
		return "", fmt.Errorf("target url scheme in mapping (%s) and in query (%s) are not equal", expectedScheme, targetUrl.Scheme)
	}

	targetUrl.Host = r.target.host
	if len(r.target.scheme) > 0 {
		targetUrl.Scheme = r.target.scheme
	}

	return targetUrl.String(), nil
}
