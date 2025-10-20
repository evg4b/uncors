package config

import (
	"net/url"
	"reflect"

	"github.com/evg4b/uncors/internal/urlparser"
	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"
)

type Mapping struct {
	From            string            `mapstructure:"from"`
	To              string            `mapstructure:"to"`
	Statics         StaticDirectories `mapstructure:"statics"`
	Mocks           Mocks             `mapstructure:"mocks"`
	Scripts         Scripts           `mapstructure:"scripts"`
	Cache           CacheGlobs        `mapstructure:"cache"`
	Rewrites        RewriteOptions    `mapstructure:"rewrites"`
	OptionsHandling OptionsHandling   `mapstructure:"options-handling"`

	// Cached parsed URL and its components (not serialized)
	fromURL  *url.URL `json:"-" mapstructure:"-" yaml:"-"`
	fromHost string   `json:"-" mapstructure:"-" yaml:"-"`
	fromPort string   `json:"-" mapstructure:"-" yaml:"-"`
}

func (m *Mapping) Clone() Mapping {
	return Mapping{
		From:            m.From,
		To:              m.To,
		Statics:         m.Statics.Clone(),
		Mocks:           m.Mocks.Clone(),
		Scripts:         m.Scripts.Clone(),
		Cache:           m.Cache.Clone(),
		Rewrites:        m.Rewrites.Clone(),
		OptionsHandling: m.OptionsHandling.Clone(),
		fromURL:         m.fromURL, // Share cached URL
		fromHost:        m.fromHost,
		fromPort:        m.fromPort,
	}
}

// GetFromURL returns the parsed URL, caching it on first access.
// This method performs lazy parsing to avoid redundant URL parsing operations.
func (m *Mapping) GetFromURL() (*url.URL, error) {
	if m.fromURL == nil {
		parsedURL, err := urlparser.Parse(m.From)
		if err != nil {
			return nil, err
		}
		m.fromURL = parsedURL
	}

	return m.fromURL, nil
}

// GetFromHostPort returns the host and port from the From URL, caching them on first access.
// This method combines URL parsing and host/port splitting to avoid redundant operations.
func (m *Mapping) GetFromHostPort() (string, string, error) {
	if m.fromHost == "" && m.fromPort == "" {
		uri, err := m.GetFromURL()
		if err != nil {
			return "", "", err
		}
		m.fromHost, m.fromPort, err = urlparser.SplitHostPort(uri)
		if err != nil {
			return "", "", err
		}
	}

	return m.fromHost, m.fromPort, nil
}

// ClearCache clears the cached URL and its components. This is primarily used for testing.
func (m *Mapping) ClearCache() {
	m.fromURL = nil
	m.fromHost = ""
	m.fromPort = ""
}

var (
	mappingType   = reflect.TypeOf(Mapping{})
	mappingFields = getTagValues(mappingType, "mapstructure")
)

func URLMappingHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, rawData any) (any, error) {
		if t != mappingType || f.Kind() != reflect.Map {
			return rawData, nil
		}

		if data, ok := rawData.(map[string]any); ok {
			availableFields, _ := lo.Difference(lo.Keys(data), mappingFields)
			if len(data) == 1 && len(availableFields) == 1 {
				return Mapping{
					From: availableFields[0],
					To:   data[availableFields[0]].(string), // nolint: forcetypeassert
				}, nil
			}

			mapping := Mapping{}
			err := decodeConfig(
				data,
				&mapping,
				StaticDirMappingHookFunc(),
			)

			return mapping, err
		}

		return rawData, nil
	}
}

func getTagValues(typeValue reflect.Type, tag string) []string {
	fields := reflect.VisibleFields(typeValue)

	return lo.FilterMap(fields, func(field reflect.StructField, _ int) (string, bool) {
		return field.Tag.Lookup(tag)
	})
}
