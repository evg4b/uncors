package urlreplacer

import (
	"errors"
	"fmt"
	"github.com/evg4b/uncors/pkg/urlglob"
	"github.com/evg4b/uncors/pkg/urlx"
	"net/url"
)

type urlMappingV2 struct {
	rawSource string
	source    *ReplacerV2
	rawTarget string
	target    *ReplacerV2
}

type URLReplacerFactory struct { // nolint: revive
	mappingsV2 []urlMappingV2
}

var ErrMappingNotFound = errors.New("mapping not found")
var ErrMappingNotSpecified = errors.New("you must specify at least one mapping")

func NewURLReplacerFactory(mappings map[string]string) (*URLReplacerFactory, error) {
	if len(mappings) < 1 {
		return nil, ErrMappingNotSpecified
	}

	urlMappingsV2 := []urlMappingV2{}
	for sourceURL, targetURL := range mappings {
		sourceGlob, err := urlglob.NewURLGlob(sourceURL)
		if err != nil {
			return nil, fmt.Errorf("failed to configure mappings: %w", err)
		}

		targetGlob, err := urlglob.NewURLGlob(targetURL)
		if err != nil {
			return nil, fmt.Errorf("failed to configure mappings: %w", err)
		}

		if sourceGlob.WildCardCount != targetGlob.WildCardCount {
			return nil, urlglob.ErrTooManyWildcards
		}

		parsedSource, err := urlx.Parse(sourceURL)
		if err != nil {
			return nil, fmt.Errorf("failed to configure mappings: %w", err)
		}

		target, source, err := makeV2(parsedSource.String(), targetURL)
		if err != nil {
			return nil, fmt.Errorf("failed to configure mappings: %w", err)
		}

		urlMappingsV2 = append(urlMappingsV2, urlMappingV2{
			rawSource: sourceURL,
			source:    source,
			rawTarget: targetURL,
			target:    target,
		})
	}

	return &URLReplacerFactory{urlMappingsV2}, nil
}

func (f *URLReplacerFactory) MakeV2(requestURL *url.URL) (*ReplacerV2, *ReplacerV2, error) {
	mapping, err := f.findMappingV2(requestURL.String())
	if err != nil {
		return nil, nil, err
	}

	return mapping.target, mapping.source, nil
}

func makeV2(rawSource, rawTarget string) (*ReplacerV2, *ReplacerV2, error) {
	target, err := NewReplacerV2(rawSource, rawTarget)
	if err != nil {
		return nil, nil, err
	}

	source, err := NewReplacerV2(rawTarget, rawSource)
	if err != nil {
		return nil, nil, err
	}

	return target, source, nil
}

func (f *URLReplacerFactory) findMappingV2(requestURL string) (urlMappingV2, error) {
	for _, imapping := range f.mappingsV2 {
		if imapping.target.IsMatched(requestURL) {
			return imapping, nil
		}
	}

	return urlMappingV2{}, ErrMappingNotFound
}
