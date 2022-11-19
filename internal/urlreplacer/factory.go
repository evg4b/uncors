package urlreplacer

import (
	"errors"
	"fmt"
	"net/url"
)

type mapping struct {
	rawSource string
	source    *Replacer
	rawTarget string
	target    *Replacer
}

type Factory struct {
	mappings []mapping
}

var (
	ErrMappingNotFound     = errors.New("mapping not found")
	ErrMappingNotSpecified = errors.New("you must specify at least one mapping")
)

func NewURLReplacerFactory(urlMappings map[string]string) (*Factory, error) {
	if len(urlMappings) < 1 {
		return nil, ErrMappingNotSpecified
	}

	var mappings []mapping //nolint:prealloc
	for sourceURL, targetURL := range urlMappings {
		target, source, err := replacers(sourceURL, targetURL)
		if err != nil {
			return nil, fmt.Errorf("failed to configure url mappings: %w", err)
		}

		mappings = append(mappings, mapping{
			rawSource: sourceURL,
			source:    source,
			rawTarget: targetURL,
			target:    target,
		})
	}

	return &Factory{mappings}, nil
}

func (f *Factory) Make(requestURL *url.URL) (*Replacer, *Replacer, error) {
	mapping, err := f.findMapping(requestURL.String())
	if err != nil {
		return nil, nil, err
	}

	return mapping.target, mapping.source, nil
}

func replacers(rawSource, rawTarget string) (*Replacer, *Replacer, error) {
	target, err := NewReplacer(rawSource, rawTarget)
	if err != nil {
		return nil, nil, err
	}

	source, err := NewReplacer(rawTarget, rawSource)
	if err != nil {
		return nil, nil, err
	}

	return target, source, nil
}

func (f *Factory) findMapping(requestURL string) (mapping, error) {
	for _, mapping := range f.mappings {
		if mapping.target.IsMatched(requestURL) {
			return mapping, nil
		}
	}

	return mapping{}, ErrMappingNotFound
}
