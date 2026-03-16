package config

import (
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// HARConfig defines settings for the HAR (HTTP Archive) collector middleware.
// When File is non-empty, all requests/responses passing through the proxy
// for this mapping will be recorded to the specified HAR file.
type HARConfig struct {
	File                 string `mapstructure:"file"`
	CaptureSecureHeaders bool   `mapstructure:"capture-secure-headers"`
}

func (h HARConfig) Enabled() bool {
	return h.File != ""
}

func (h HARConfig) Clone() HARConfig {
	return HARConfig{
		File:                 h.File,
		CaptureSecureHeaders: h.CaptureSecureHeaders,
	}
}

var harConfigType = reflect.TypeFor[HARConfig]()

// HARConfigHookFunc returns a mapstructure decode hook that allows HARConfig
// to be specified as a plain string in YAML/config files.
//
// Short form:  har: ./recordings/api.har
// Full form:   har: { file: ./recordings/api.har, capture-secure-headers: true }
func HARConfigHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, rawData any) (any, error) {
		if t != harConfigType || f.Kind() != reflect.String {
			return rawData, nil
		}

		if file, ok := rawData.(string); ok {
			return HARConfig{File: file}, nil
		}

		return rawData, nil
	}
}
