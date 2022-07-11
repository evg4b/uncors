package urlreplacer

import (
	"fmt"
	"net/url"
)

type urlMepping struct {
	source *url.URL
	target *url.URL
}

type SimpleReplacer struct {
	mappings map[string]urlMepping
}

func NewSimpleReplacer(mapping map[string]string) *SimpleReplacer {
	mappings := map[string]urlMepping{}

	for sourceUrl, targetUrl := range mapping {
		source, err := url.Parse(sourceUrl)
		if err != nil {
			panic(err)
		}

		target, err := url.Parse(targetUrl)
		if err != nil {
			panic(err)
		}

		mappings[source.Host] = urlMepping{source, target}
	}

	return &SimpleReplacer{mappings}
}

func sourceUrlGetter(mapping urlMepping) *url.URL {
	return mapping.source
}

func targetUrlGetter(mapping urlMepping) *url.URL {
	return mapping.target
}

func (r *SimpleReplacer) ToTarget(targetUrl string) (string, error) {
	parsedUrl, err := url.Parse(targetUrl)
	if err != nil {
		return "", fmt.Errorf("failed parse url: '%s': %v", targetUrl, err)
	}

	return r.transformUrl(parsedUrl, parsedUrl.Host, targetUrlGetter)
}

func (r *SimpleReplacer) ToSource(targetUrl string, host string) (string, error) {
	parsedUrl, err := url.Parse(targetUrl)
	if err != nil {
		return "", fmt.Errorf("failed parse url: '%s': %v", targetUrl, err)
	}

	return r.transformUrl(parsedUrl, host, sourceUrlGetter)
}

func (r *SimpleReplacer) transformUrl(current *url.URL, host string, getter func(mapping urlMepping) *url.URL) (string, error) {
	urlMapping, ok := r.mappings[host]
	if !ok {
		return "", fmt.Errorf("failed to find mapping for host '%s'", current.Host)
	}

	target := getter(urlMapping)

	if len(target.Scheme) > 0 {
		current.Scheme = target.Scheme
	}

	current.Host = target.Host

	return current.String(), nil
}
