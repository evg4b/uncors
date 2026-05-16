package config

import (
	"fmt"
	"path/filepath"

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

func (h *HARConfig) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		h.File = value.Value

		return nil
	}

	type harConfigAlias HARConfig

	return value.Decode((*harConfigAlias)(h))
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
