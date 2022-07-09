package urlreplacer

import (
	"fmt"
	"net/url"
)

type SimpleReplacer struct {
	source *url.URL
	target *url.URL
}

func NewSimpleReplacer(source string, target string) *SimpleReplacer {
	sourceUri, err := url.Parse(source)
	if err != nil {
		panic(err)
	}

	targetUri, err := url.Parse(target)
	if err != nil {
		panic(err)
	}

	return &SimpleReplacer{
		source: sourceUri,
		target: targetUri,
	}
}

func (r *SimpleReplacer) ToTarget(targetUrl string) (string, error) {
	return transformUrl(targetUrl, r.target)
}

func (r *SimpleReplacer) ToSource(targetUrl string) (string, error) {
	return transformUrl(targetUrl, r.source)
}

func transformUrl(targetUrl string, target *url.URL) (string, error) {
	parsedUrl, err := url.Parse(targetUrl)
	if err != nil {
		return "", fmt.Errorf("failed parse url: '%s': %v", target, err)
	}

	parsedUrl.Scheme = target.Scheme
	parsedUrl.Host = target.Host

	return parsedUrl.String(), nil
}
