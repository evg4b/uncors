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

	for source, target := range mapping {
		sourceUri, err := url.Parse(source)
		if err != nil {
			panic(err)
		}

		targetUri, err := url.Parse(target)
		if err != nil {
			panic(err)
		}

		mappings[sourceUri.Host] = urlMepping{
			source: sourceUri,
			target: targetUri,
		}

	}

	return &SimpleReplacer{
		mappings: mappings,
	}
}

func sourceUrlGetter(mapping urlMepping) *url.URL {
	return mapping.source
}

func targetUrlGetter(mapping urlMepping) *url.URL {
	return mapping.source
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

func (r *SimpleReplacer) transformUrl(target *url.URL, host string, getter func(mapping urlMepping) *url.URL) (string, error) {

	urtMapping, ok := r.mappings[host]
	if !ok {
		return "", fmt.Errorf("failed to find mapping for host '%s'", target.Host)
	}

	targetSource := getter(urtMapping)

	if urtMapping.source.Scheme != targetSource.Scheme {
		return "", fmt.Errorf("failed to find mapping for scheme '%s' and host '%s'", target.Scheme, target.Host)
	}

	target.Scheme = targetSource.Scheme
	target.Host = targetSource.Host

	return target.String(), nil
}
