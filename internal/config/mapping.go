package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/evg4b/uncors/internal/urlparser"
	"github.com/spf13/afero"
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

func ValidateProxy(field, value string, errs *Errors) {
	if value == "" {
		return
	}

	_, err := urlparser.Parse(value)
	if err != nil {
		errs.add(fmt.Sprintf("%s is not a valid URL", field))
	}
}

func ValidateCacheGlob(field, value string, errs *Errors) {
	ValidateGlobPattern(field, value, errs)
}

func ValidateTLS(_ string, mapping Mapping, fs afero.Fs, errs *Errors) {
	fromURL, err := mapping.GetFromURL()
	if err != nil || fromURL.Scheme != "https" {
		return
	}

	if !infratls.CAExists(fs) {
		errs.add(formatTLSError(fromURL.Host))
	}
}

func formatTLSError(host string) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "HTTPS mapping '%s' requires a local CA certificate for automatic TLS.\n\n", host)
	builder.WriteString("Generate a local CA certificate:\n")
	builder.WriteString("  uncors generate-certs\n\n")
	builder.WriteString("After generating CA, you can add it to your system's trusted certificates.")

	return builder.String()
}

func (m Mapping) Validate(field string, fs afero.Fs, errs *Errors) {
	ValidateHost(joinPath(field, "from"), m.From, errs)
	ValidateHost(joinPath(field, "to"), m.To, errs)
	m.OptionsHandling.Validate(joinPath(field, "options-handling"), errs)
	m.HAR.Validate(joinPath(field, "har"), errs)
	ValidateTLS(field, m, fs, errs)

	for i, static := range m.Statics {
		static.Validate(joinPath(field, "statics", index(i)), fs, errs)
	}

	for i, mock := range m.Mocks {
		mock.Validate(joinPath(field, "mocks", index(i)), fs, errs)
	}

	for i, glob := range m.Cache {
		ValidateCacheGlob(joinPath(field, "cache", index(i)), glob, errs)
	}

	for i, rewrite := range m.Rewrites {
		rewrite.Validate(joinPath(field, "rewrite", index(i)), errs)
	}

	for i, script := range m.Scripts {
		script.Validate(joinPath(field, "scripts", index(i)), fs, errs)
	}
}
