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
		parsedURL, err := urlparser.Parse(m.From)
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

		m.fromHost, m.fromPort, err = urlparser.SplitHostPort(uri)
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
	if err != nil || fromURL.Scheme != httpsScheme {
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

func (m *Mapping) Validate(field string, fs afero.Fs, errs *Errors) {
	ValidateHost(joinPath(field, "from"), m.From, errs)
	ValidateHost(joinPath(field, "to"), m.To, errs)
	m.OptionsHandling.Validate(joinPath(field, "options-handling"), errs)
	m.HAR.Validate(joinPath(field, "har"), errs)
	ValidateTLS(field, *m, fs, errs)

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
