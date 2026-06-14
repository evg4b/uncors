package config

import (
	"errors"
	"fmt"

	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

var ErrMappingShorthandValue = errors.New("mapping shorthand value must be a string URL")

type Mapping struct {
	From            urlt.Host         `yaml:"from"`
	To              urlt.Host         `yaml:"to"`
	Statics         StaticDirectories `yaml:"statics"`
	Mocks           Mocks             `yaml:"mocks"`
	Scripts         Scripts           `yaml:"scripts"`
	Cache           CacheGlobs        `yaml:"cache"`
	Rewrites        RewriteOptions    `yaml:"rewrites"`
	OptionsHandling OptionsHandling   `yaml:"options-handling"`
	HAR             HARConfig         `yaml:"har"`
}

var knownMappingFields = map[string]bool{
	"from": true, "to": true, "statics": true, "mocks": true,
	"scripts": true, "cache": true, "rewrites": true,
	"options-handling": true, "har": true,
}

func (m *Mapping) UnmarshalYAML(value *yaml.Node) error {
	if isShorthandMapping(value) {
		return m.unmarshalShorthand(value)
	}

	type mappingAlias Mapping

	return value.Decode((*mappingAlias)(m))
}

// isShorthandMapping reports whether the node uses the `from: to` shorthand
// form (a single key that is not one of the known mapping fields).
func isShorthandMapping(value *yaml.Node) bool {
	if value.Kind != yaml.MappingNode || len(value.Content) != 2 {
		return false
	}

	return !knownMappingFields[value.Content[0].Value]
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
	}
}

func ValidateProxy(field, value string) error {
	if value == "" {
		return nil
	}

	_, err := parseLooseURL(value)
	if err != nil {
		return &ValidationError{fmt.Sprintf("%s is not a valid URL", field)}
	}

	return nil
}

func ValidateTLS(_ string, mapping Mapping, fs afero.Fs) error {
	if mapping.From.Scheme != httpsScheme {
		return nil
	}

	if !server.CAExists(fs) {
		return &TLSError{mapping.From.HostPort()}
	}

	return nil
}

func (m *Mapping) Validate(field string, fs afero.Fs) error {
	errs := make([]error, 0, 5+len(m.Statics)+len(m.Mocks)+len(m.Cache)+len(m.Rewrites)+len(m.Scripts))

	errs = append(errs, ValidateHost(joinPath(field, "from"), m.From.String()))
	errs = append(errs, ValidateHost(joinPath(field, "to"), m.To.String()))
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

func (m *Mapping) unmarshalShorthand(value *yaml.Node) error {
	if value.Content[1].Tag != "!!str" {
		return ErrMappingShorthandValue
	}

	from, err := urlt.ParseHost(value.Content[0].Value)
	if err != nil {
		return err
	}

	to, err := urlt.ParseHost(value.Content[1].Value)
	if err != nil {
		return err
	}

	m.From = *from
	m.To = *to

	return nil
}
