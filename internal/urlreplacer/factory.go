package urlreplacer

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/evg4b/uncors/pkg/urlglob"
)

type urlMapping struct {
	rawSource  string
	sourceGlob *urlglob.URLGlob
	rawTarget  string
	targetGlob *urlglob.URLGlob
}

type URLReplacerFactory struct { // nolint: revive
	mappings []urlMapping
}

var ErrMappingNotFound = errors.New("mapping not found")
var ErrMappingNotSpecified = errors.New("you must specify at least one mapping")

func NewURLReplacerFactory(mappings map[string]string) (*URLReplacerFactory, error) {
	if len(mappings) < 1 {
		return nil, ErrMappingNotSpecified
	}

	urlMappings := []urlMapping{}
	for sourceURL, targetURL := range mappings {
		sourceGlob, err := urlglob.NewURLGlob(sourceURL)
		if err != nil {
			return nil, fmt.Errorf("failed to configure mappings: %w", err)
		}

		targetGlob, err := urlglob.NewURLGlob(targetURL, urlglob.SaveOriginalPort())
		if err != nil {
			return nil, fmt.Errorf("failed to configure mappings: %w", err)
		}

		if sourceGlob.WildCardCount != targetGlob.WildCardCount {
			return nil, urlglob.ErrTooManyWildcards
		}

		urlMappings = append(urlMappings, urlMapping{
			rawSource:  sourceURL,
			sourceGlob: sourceGlob,
			rawTarget:  targetURL,
			targetGlob: targetGlob,
		})
	}

	return &URLReplacerFactory{urlMappings}, nil
}

func (f *URLReplacerFactory) Make(requestURL *url.URL) (*Replacer, error) {
	mapping := f.findMapping(requestURL)

	if mapping == nil || (len(mapping.sourceGlob.Scheme) > 0 && mapping.sourceGlob.Scheme != requestURL.Scheme) {
		return nil, ErrMappingNotFound
	}

	return &Replacer{
		rawSource: mapping.rawSource,
		source:    mapping.sourceGlob,
		rawTarget: mapping.rawTarget,
		target:    mapping.targetGlob,
	}, nil
}

func (f *URLReplacerFactory) findMapping(requestURL *url.URL) *urlMapping {
	for _, imapping := range f.mappings {
		if imapping.sourceGlob.Match(requestURL) {
			return &imapping
		}
	}

	return nil
}
