package config

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

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

	fromURL  *url.URL `yaml:"-"`
	fromHost string   `yaml:"-"`
	fromPort string   `yaml:"-"`
}

var knownMappingFields = map[string]bool{
	"from": true, "to": true, "statics": true, "mocks": true,
	"scripts": true, "cache": true, "rewrites": true,
	"options-handling": true, "har": true,
}

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

func (m *Mapping) GetFromURL() (*url.URL, error) {
	if m.fromURL == nil {
		parsedURL, err := urlt.Parse(m.From)
		if err != nil {
			return nil, err
		}

		m.fromURL = parsedURL
	}

	return m.fromURL, nil
}

func (m *Mapping) GetFromHostPort() (string, string, error) {
	if m.fromHost == "" && m.fromPort == "" {
		uri, err := m.GetFromURL()
		if err != nil {
			return "", "", err
		}

		m.fromHost, m.fromPort, err = urlt.SplitHostPort(uri)
		if err != nil {
			return "", "", err
		}
	}

	return m.fromHost, m.fromPort, nil
}

func (m *Mapping) ClearCache() {
	m.fromURL = nil
	m.fromHost = ""
	m.fromPort = ""
}

func ValidateProxy(field, value string) error {
	if value == "" {
		return nil
	}

	_, err := urlt.Parse(value)
	if err != nil {
		return &ValidationError{fmt.Sprintf("%s is not a valid URL", field)}
	}

	return nil
}

func ValidateTLS(_ string, mapping Mapping, fs afero.Fs) error {
	fromURL, err := mapping.GetFromURL()
	if err != nil {
		return nil //nolint:nilerr
	}

	if fromURL.Scheme != httpsScheme {
		return nil
	}

	if !server.CAExists(fs) {
		return &TLSError{fromURL.Host}
	}

	return nil
}

func (m *Mapping) Validate(field string, fs afero.Fs) error {
	errs := make([]error, 0, 5+len(m.Statics)+len(m.Mocks)+len(m.Cache)+len(m.Rewrites)+len(m.Scripts))

	errs = append(errs, ValidateHost(joinPath(field, "from"), m.From))
	errs = append(errs, ValidateHost(joinPath(field, "to"), m.To))
	errs = append(errs, m.OptionsHandling.Validate(joinPath(field, "options-handling")))
	errs = append(errs, m.HAR.Validate(joinPath(field, "har")))
	errs = append(errs, ValidateTLS(field, *m, fs))

	for i, static := range m.Statics {
		errs = append(errs, static.Validate(joinPath(field, "statics", index(i)), fs))
	}

	for i, mock := range m.Mocks {
		errs = append(errs, mock.Validate(joinPath(field, "mocks", index(i)), fs))
	}

	for i, glob := range m.Cache {
		errs = append(errs, ValidateGlobPattern(joinPath(field, "cache", index(i)), glob))
	}

	for i, rewrite := range m.Rewrites {
		errs = append(errs, rewrite.Validate(joinPath(field, "rewrite", index(i))))
	}

	for i, script := range m.Scripts {
		errs = append(errs, script.Validate(joinPath(field, "scripts", index(i)), fs))
	}

	return errors.Join(errs...)
}
