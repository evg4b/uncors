package urlreplacer

import (
	"errors"
	"fmt"
	"github.com/evg4b/uncors/pkg/urlglob"
	"github.com/evg4b/uncors/pkg/urlx"
	"net/url"
)

type mapping struct {
	rawSource string
	source    *Replacer
	rawTarget string
	target    *Replacer
}

type Factory struct { // nolint: revive
	mappings []mapping
}

var ErrMappingNotFound = errors.New("mapping not found")
var ErrMappingNotSpecified = errors.New("you must specify at least one mapping")

func NewURLReplacerFactory(urlMappings map[string]string) (*Factory, error) {
	if len(urlMappings) < 1 {
		return nil, ErrMappingNotSpecified
	}

	var mappings []mapping
	for sourceURL, targetURL := range urlMappings {
		sourceGlob, err := urlglob.NewURLGlob(sourceURL)
		if err != nil {
			return nil, fmt.Errorf("failed to configure urlMappings: %w", err)
		}

		targetGlob, err := urlglob.NewURLGlob(targetURL)
		if err != nil {
			return nil, fmt.Errorf("failed to configure urlMappings: %w", err)
		}

		if sourceGlob.WildCardCount != targetGlob.WildCardCount {
			return nil, urlglob.ErrTooManyWildcards
		}

		parsedSource, err := urlx.Parse(sourceURL)
		if err != nil {
			return nil, fmt.Errorf("failed to configure urlMappings: %w", err)
		}

		target, source, err := replacer(parsedSource.String(), targetURL)
		if err != nil {
			return nil, fmt.Errorf("failed to configure urlMappings: %w", err)
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

func replacer(rawSource, rawTarget string) (*Replacer, *Replacer, error) {
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
