package config

import "gopkg.in/yaml.v3"

// HARConfig defines settings for the HAR (HTTP Archive) collector middleware.
// When File is non-empty, all requests/responses passing through the proxy
// for this mapping will be recorded to the specified HAR file.
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

// UnmarshalYAML allows HARConfig to be specified as a plain string (file path)
// or as a full mapping.
//
// Short form:  har: ./recordings/api.har
// Full form:   har: { file: ./recordings/api.har, capture-secure-headers: true }.
func (h *HARConfig) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		h.File = value.Value

		return nil
	}

	type harConfigAlias HARConfig

	return value.Decode((*harConfigAlias)(h))
}
