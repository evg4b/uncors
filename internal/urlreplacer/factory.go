package urlreplacer

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

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
		sourceHasTLS:         isSourceSecure(requestURL),
		rawTarget:            mapping.rawTarget,
		target:               mapping.targetGlob,
		targetReplacePattern: mapping.targetReplacePattern,
		targetHasTLS:         isTargetSecure(mapping, requestURL),
	}, nil
}

func (f *URLReplacerFactory) MakeV2(requestURL *url.URL) (*ReplacerV2, error) {
	return &ReplacerV2{}, nil
}

func isTargetSecure(mapping urlMapping, requestURL *url.URL) bool {
	if strings.EqualFold(mapping.targetGlob.Scheme, "https") {
		return true
	}

	return len(mapping.targetGlob.Scheme) == 0 && strings.EqualFold(requestURL.Scheme, "https")
}

func isSourceSecure(requestURL *url.URL) bool {
	return strings.EqualFold(requestURL.Scheme, "https")
}

func (f *URLReplacerFactory) findMapping(requestURL *url.URL) (urlMapping, error) {
	for _, mapping := range f.mappings {
		if mapping.sourceGlob.Match(requestURL) {
			return mapping, nil
		}
	}

	return urlMapping{}, ErrMappingNotFound
}
