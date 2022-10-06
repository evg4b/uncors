package urlreplacer

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/evg4b/uncors/pkg/urlglob"
	"github.com/evg4b/uncors/pkg/urlx"
)

type urlMapping struct {
	rawSource            string
	sourceGlob           *urlglob.URLGlob
	sourceReplacePattern urlglob.ReplacePattern
	rawTarget            string
	targetGlob           *urlglob.URLGlob
	targetReplacePattern urlglob.ReplacePattern
}

type urlMappingV2 struct {
	rawSource string
	source    *ReplacerV2
	rawTarget string
	target    *ReplacerV2
}

type URLReplacerFactory struct { // nolint: revive
	mappings   []urlMapping
	mappingsV2 []urlMappingV2
}

var ErrMappingNotFound = errors.New("mapping not found")
var ErrMappingNotSpecified = errors.New("you must specify at least one mapping")

func NewURLReplacerFactory(mappings map[string]string) (*URLReplacerFactory, error) {
	if len(mappings) < 1 {
		return nil, ErrMappingNotSpecified
	}

	urlMappings := []urlMapping{}
	urlMappingsV2 := []urlMappingV2{}
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

	return &URLReplacerFactory{
		urlMappings,
		urlMappingsV2,
	}, nil
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

func (f *URLReplacerFactory) findMappingV2(requestURL string) (urlMappingV2, error) {
	for _, imapping := range f.mappingsV2 {
		if imapping.target.IsMatched(requestURL) {
			return imapping, nil
		}
	}

	return urlMappingV2{}, ErrMappingNotFound
}
