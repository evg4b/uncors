package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"

	"gopkg.in/yaml.v3"
)

type HARConfig struct {
	File                 string `yaml:"file"`
	CaptureSecureHeaders bool   `yaml:"capture-secure-headers"`
}

func (h *HARConfig) Enabled() bool {
	return h.File != ""
}

func (h *HARConfig) Clone() HARConfig {
	return HARConfig{
		File:                 h.File,
		CaptureSecureHeaders: h.CaptureSecureHeaders,
	}
}

type harConfigAlias HARConfig

func (h *HARConfig) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		h.File = value.Value

		return nil
	}

	return value.Decode((*harConfigAlias)(h))
}

func (h *HARConfig) checkKnownFields(field string, node *yaml.Node) error {
	return checkStructFields(field, node, reflect.TypeFor[harConfigAlias]())
}

func (h *HARConfig) Validate(field string) error {
	if !h.Enabled() {
		return nil
	}

	if filepath.Ext(h.File) == "" {
		return &ValidationError{fmt.Sprintf("%s: HAR file path %q must have a file extension (e.g. .har)", field, h.File)}
	}

	return nil
}

func validateUniqueHARFiles(mappings Mappings) error {
	owners := make(map[string]string, len(mappings))

	var errs []error

	for i, mapping := range mappings {
		if !mapping.HAR.Enabled() {
			continue
		}

		field := joinPath("mappings", index(i), "har")
		key := harFileKey(mapping.HAR.File)

		if owner, ok := owners[key]; ok {
			errs = append(errs, &ValidationError{
				fmt.Sprintf("%s: HAR file %q is already used by %s", field, mapping.HAR.File, owner),
			})

			continue
		}

		owners[key] = field
	}

	return errors.Join(errs...)
}

func harFileKey(file string) string {
	abs, err := filepath.Abs(file)
	if err != nil {
		return filepath.Clean(file)
	}

	return abs
}
