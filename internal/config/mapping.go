package config

import (
	"errors"
	"net/url"

	"github.com/evg4b/uncors/internal/urlparser"
	"gopkg.in/yaml.v3"
)

// ErrMappingShorthandValue is returned when a URL shorthand mapping has a
// non-string value (e.g. "http://localhost: 123" instead of a URL string).
var ErrMappingShorthandValue = errors.New("mapping shorthand value must be a string URL")

type Mapping struct {
	From            string            `yaml:"from"`
	To              string            `yaml:"to"`
	Statics         StaticDirectories `yaml:"statics"`
	Mocks           Mocks             `yaml:"mocks"`
	Scripts         Scripts           `yaml:"scripts"`
	Cache           CacheGlobs        `yaml:"cache"`
	Rewrites        RewriteOptions    `yaml:"rewrites"`
	OptionsHandling OptionsHandling   `yaml:"options-handling"`
	HAR             HARConfig         `yaml:"har"`

	// Cached parsed URL and its components (not serialized)
	fromURL  *url.URL `yaml:"-"`
	fromHost string   `yaml:"-"`
	fromPort string   `yaml:"-"`
}

// knownMappingFields is the set of yaml keys that belong to a full Mapping
// object. Any single-key YAML map whose key is NOT in this set is interpreted
// as the shorthand "from: to" form.
var knownMappingFields = map[string]bool{
	"from": true, "to": true, "statics": true, "mocks": true,
	"scripts": true, "cache": true, "rewrites": true,
	"options-handling": true, "har": true,
}

// UnmarshalYAML decodes a Mapping from YAML. It recognises two forms:
//
// Shorthand — a single-key mapping whose key is not a known field name:
//
//	http://localhost:8080: https://example.com
//
// Full form — a standard YAML mapping with "from", "to", and optional fields.
func (m *Mapping) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.MappingNode && len(value.Content) == 2 {
		key := value.Content[0].Value
		if !knownMappingFields[key] {
			if value.Content[1].Tag != "!!str" {
				return ErrMappingShorthandValue
			}

			m.From = key
			m.To = value.Content[1].Value

			return nil
		}
	}

	type mappingAlias Mapping

	return value.Decode((*mappingAlias)(m))
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
		HAR:             m.HAR.Clone(),
		fromURL:         m.fromURL,
		fromHost:        m.fromHost,
		fromPort:        m.fromPort,
	}
}

// GetFromURL returns the parsed URL, caching it on first access.
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
