package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"strings"

	"github.com/evg4b/uncors/internal/certs"
	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

const ListenAddress = "127.0.0.1"

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
	AllowedOrigins  AllowedOrigins    `yaml:"allowed-origins"`
	HAR             HARConfig         `yaml:"har"`
}

type mappingAlias Mapping

func (m *Mapping) UnmarshalYAML(value *yaml.Node) error {
	if isShorthandMapping(value) {
		return m.unmarshalShorthand(value)
	}

	return value.Decode((*mappingAlias)(m))
}

func isShorthandMapping(value *yaml.Node) bool {
	if value.Kind != yaml.MappingNode || len(value.Content) != 2 {
		return false
	}

	key := value.Content[0].Value

	return strings.ContainsAny(key, ".:") || strings.EqualFold(key, "localhost")
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
		AllowedOrigins:  m.AllowedOrigins.Clone(),
		HAR:             m.HAR.Clone(),
	}
}

func ValidateProxy(field, value string) error {
	if value == "" {
		return nil
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return &ValidationError{fmt.Sprintf("%s is not a valid URL", field)}
	}

	return nil
}

func ValidateTLS(mapping Mapping, fs afero.Fs) error {
	if mapping.From.Scheme != httpsScheme {
		return nil
	}

	if !certs.Exists(fs) {
		return &TLSError{mapping.From.HostPort()}
	}

	return nil
}

func validateListenIP(field string, from urlt.Host) error {
	ip := net.ParseIP(from.Hostname)
	if ip == nil || ip.Equal(net.ParseIP(ListenAddress)) || ip.Equal(net.IPv4zero) {
		return nil
	}

	return &ValidationError{fmt.Sprintf(
		"%s: UNCORS listens on %s only, so %s is never reached; use %s or a host name",
		field, ListenAddress, from.Hostname, ListenAddress,
	)}
}

func (m *Mapping) Validate(field string, fs afero.Fs) error {
	errs := make([]error, 0, 7+len(m.Statics)+len(m.Mocks)+len(m.Cache)+len(m.Rewrites)+len(m.Scripts))

	errs = append(errs, ValidateHost(joinPath(field, "from"), m.From))
	errs = append(errs, validateListenIP(joinPath(field, "from"), m.From))
	errs = append(errs, ValidateHost(joinPath(field, "to"), m.To))
	errs = append(errs, m.OptionsHandling.Validate(joinPath(field, "options-handling")))
	errs = append(errs, m.AllowedOrigins.Validate(joinPath(field, "allowed-origins")))
	errs = append(errs, m.HAR.Validate(joinPath(field, "har")))
	errs = append(errs, ValidateTLS(*m, fs))

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
		errs = append(errs, rewrite.Validate(joinPath(field, "rewrites", index(i))))
	}

	for i, script := range m.Scripts {
		errs = append(errs, script.Validate(joinPath(field, "scripts", index(i)), fs))
	}

	return errors.Join(errs...)
}

func (m *Mapping) checkKnownFields(field string, node *yaml.Node) error {
	if isShorthandMapping(node) {
		return nil
	}

	return checkStructFields(field, node, reflect.TypeFor[mappingAlias]())
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
