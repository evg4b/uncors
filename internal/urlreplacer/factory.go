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

type URLReplacerFactory struct { // nolint: revive
	mappings map[string]urlMapping
}

var ErrMappingNotFound = errors.New("mapping not found")
var ErrMappingNotSpecified = errors.New("you must specify at least one mapping")

func NewURLReplacerFactory(mappings map[string]string) (*URLReplacerFactory, error) {
	if len(mappings) < 1 {
		return nil, ErrMappingNotSpecified
	}

	urlMappings := map[string]urlMapping{}
	for sourceURL, targetURL := range mappings {
		source, err := parseSourceURL(sourceURL)
		if err != nil {
			return nil, err
		}

		target, err := parseTargetURL(targetURL)
		if err != nil {
			return nil, err
		}

		urlMappings[source.Hostname()] = urlMapping{source, target}
	}

	return &URLReplacerFactory{urlMappings}, nil
}

func (f *URLReplacerFactory) Make(requestURL *url.URL) (*Replacer, error) {
	hostname := requestURL.Hostname()
	mapping, ok := f.mappings[hostname]

	if !ok || (len(mapping.source.Scheme) > 0 && mapping.source.Scheme != requestURL.Scheme) {
		return nil, ErrMappingNotFound
	}

	return &Replacer{
		source: urlData{
			hostname: hostname,
			host:     requestURL.Host,
			scheme:   requestURL.Scheme,
		},
		target: urlData{
			hostname: mapping.target.Hostname(),
			host:     mapping.target.Host,
			scheme:   mapping.target.Scheme,
		},
	}, nil
}

func parseSourceURL(sourceURL string) (*url.URL, error) {
	source, err := url.Parse(sourceURL)
	if err != nil {
		return nil, fmt.Errorf("falied to parse source url: %w", err)
	}

	return source, nil
}

func parseTargetURL(targetURL string) (*url.URL, error) {
	source, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("falied to parse target url: %w", err)
	}

	return source, nil
}
