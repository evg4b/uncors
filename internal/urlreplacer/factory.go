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

func NewUrlReplacerFactory(mapping map[string]string) (*UrlReplacerFactory, error) {
	if len(mapping) < 1 {
		return nil, fmt.Errorf("you must specify at least one mapping")
	}

	mappings := map[string]urlMapping{}
	for sourceUrl, targetUrl := range mapping {
		source, err := validateSourceUrl(sourceUrl)
		if err != nil {
			return nil, err
		}

		target, err := validateTargetUrl(targetUrl)
		if err != nil {
			return nil, err
		}

		mappings[source.Hostname()] = urlMapping{source, target}
	}

	return &UrlReplacerFactory{mappings}, nil
}

func (f *UrlReplacerFactory) Make(requetUrl *url.URL) (*replacer, error) {
	target, ok := f.mappings[requetUrl.Hostname()]
	if !ok {
		return nil, ErrMappingNotFound
	}

	if len(target.source.Scheme) > 0 && target.source.Scheme != requetUrl.Scheme {
		return nil, ErrMappingNotFound
	}

	return &replacer{
		source: replceData{
			hostname: requetUrl.Hostname(),
			host:     requetUrl.Host,
			scheme:   requetUrl.Scheme,
		},
		target: replceData{
			hostname: target.target.Hostname(),
			host:     target.target.Host,
			scheme:   target.target.Scheme,
		},
	}, nil
}

func validateSourceUrl(sourceUrl string) (*url.URL, error) {
	source, err := url.Parse(sourceUrl)
	if err != nil {
		return nil, fmt.Errorf("falied to parse source url: %v", err)
	}

	return source, nil
}

func validateTargetUrl(targetUrl string) (*url.URL, error) {
	source, err := url.Parse(targetUrl)
	if err != nil {
		return nil, fmt.Errorf("falied to parse target url: %v", err)
	}

	return source, nil
}
