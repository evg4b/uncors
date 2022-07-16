package urlreplacer

import (
	"errors"
	"fmt"
	"net/url"
)

type urlMapping struct {
	source *url.URL
	target *url.URL
}

type UrlReplacerFactory struct {
	mappings map[string]urlMapping
}

var ErrMappingNotFound = errors.New("mapping not found")

func NewUrlReplacerFactory(mappings map[string]string) (*UrlReplacerFactory, error) {
	if len(mappings) < 1 {
		return nil, fmt.Errorf("you must specify at least one mapping")
	}

	urlMappings := map[string]urlMapping{}
	for sourceUrl, targetUrl := range mappings {
		source, err := parseSourceUrl(sourceUrl)
		if err != nil {
			return nil, err
		}

		target, err := parseTargetUrl(targetUrl)
		if err != nil {
			return nil, err
		}

		urlMappings[source.Hostname()] = urlMapping{source, target}
	}

	return &UrlReplacerFactory{urlMappings}, nil
}

func (f *UrlReplacerFactory) Make(requetUrl *url.URL) (*replacer, error) {
	hostname := requetUrl.Hostname()
	mapping, ok := f.mappings[hostname]

	if !ok || (len(mapping.source.Scheme) > 0 && mapping.source.Scheme != requetUrl.Scheme) {
		return nil, ErrMappingNotFound
	}

	return &replacer{
		source: urlData{
			hostname: hostname,
			host:     requetUrl.Host,
			scheme:   requetUrl.Scheme,
		},
		target: urlData{
			hostname: mapping.target.Hostname(),
			host:     mapping.target.Host,
			scheme:   mapping.target.Scheme,
		},
	}, nil
}

func parseSourceUrl(sourceUrl string) (*url.URL, error) {
	source, err := url.Parse(sourceUrl)
	if err != nil {
		return nil, fmt.Errorf("falied to parse source url: %v", err)
	}

	return source, nil
}

func parseTargetUrl(targetUrl string) (*url.URL, error) {
	source, err := url.Parse(targetUrl)
	if err != nil {
		return nil, fmt.Errorf("falied to parse target url: %v", err)
	}

	return source, nil
}
