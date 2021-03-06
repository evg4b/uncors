package urlreplacer

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/evg4b/uncors/pkg/urlglob"
)

type urlMapping struct {
	rawSource            string
	sourceGlob           *urlglob.URLGlob
	sourceReplacePattern urlglob.ReplacePattern
	rawTarget            string
	targetGlob           *urlglob.URLGlob
	targetReplacePattern urlglob.ReplacePattern
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

		sourceReplacePattern, err := urlglob.NewReplacePatternString(sourceURL)
		if err != nil {
			return nil, fmt.Errorf("failed to configure mappings: %w", err)
		}

		targetGlob, err := urlglob.NewURLGlob(targetURL)
		if err != nil {
			return nil, fmt.Errorf("failed to configure mappings: %w", err)
		}

		targetReplacePattern, err := urlglob.NewReplacePatternString(targetURL)
		if err != nil {
			return nil, fmt.Errorf("failed to configure mappings: %w", err)
		}

		if sourceGlob.WildCardCount != targetGlob.WildCardCount {
			return nil, urlglob.ErrTooManyWildcards
		}

		urlMappings = append(urlMappings, urlMapping{
			rawSource:            sourceURL,
			sourceGlob:           sourceGlob,
			sourceReplacePattern: sourceReplacePattern,
			rawTarget:            targetURL,
			targetGlob:           targetGlob,
			targetReplacePattern: targetReplacePattern,
		})
	}

	return &URLReplacerFactory{urlMappings}, nil
}

func (f *URLReplacerFactory) Make(requestURL *url.URL) (*Replacer, error) {
	mapping, err := f.findMapping(requestURL)
	if err != nil {
		return nil, err
	}

	if len(mapping.sourceGlob.Scheme) > 0 && mapping.sourceGlob.Scheme != requestURL.Scheme {
		return nil, ErrMappingNotFound
	}

	urlglob.PatchReplacePattern(
		&mapping.sourceReplacePattern,
		urlglob.UsePort(requestURL.Port()),
		urlglob.UseScheme(requestURL.Scheme),
	)

	return &Replacer{
		rawSource:            mapping.rawSource,
		source:               mapping.sourceGlob,
		sourceReplacePattern: mapping.sourceReplacePattern,
		rawTarget:            mapping.rawTarget,
		target:               mapping.targetGlob,
		targetReplacePattern: mapping.targetReplacePattern,
	}, nil
}

func (f *URLReplacerFactory) findMapping(requestURL *url.URL) (urlMapping, error) {
	for _, imapping := range f.mappings {
		if imapping.sourceGlob.Match(requestURL) {
			return imapping, nil
		}
	}

	return urlMapping{}, ErrMappingNotFound
}
