package config

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

// UncorsConfig is the root configuration for the uncors proxy.
type UncorsConfig struct {
	Mappings    Mappings    `mapstructure:"mappings"`
	Proxy       string      `mapstructure:"proxy"`
	Debug       bool        `mapstructure:"debug"`
	CacheConfig CacheConfig `mapstructure:"cache-config"`
	Interactive bool        `mapstructure:"interactive"`
}

// LoadConfiguration parses CLI arguments and optionally reads a YAML config file.
// CLI flags take precedence over config file values.
// Returns the loaded config, the active config file path (empty if none), and any error.
func LoadConfiguration(fs afero.Fs, args []string) (*UncorsConfig, string, error) {
	flags := defineFlags()

	err := flags.Parse(args)
	if err != nil {
		return nil, "", fmt.Errorf("failed parsing flags: %w", err)
	}

	cfg := defaultConfig()
	configPath, _ := flags.GetString("config")

	if configPath != "" {
		raw, err := readYAMLFile(fs, configPath)
		if err != nil {
			return nil, "", err
		}

		if err := applyRawConfig(raw, cfg); err != nil {
			return nil, "", fmt.Errorf("failed parsing config: %w", err)
		}
	}

	if err := applyFlagOverrides(cfg, flags); err != nil {
		return nil, "", err
	}

	cfg.Mappings = NormaliseMappings(cfg.Mappings)

	return cfg, configPath, nil
}

// readYAMLFile opens and decodes a YAML config file into a raw map.
func readYAMLFile(fs afero.Fs, path string) (map[string]any, error) {
	f, err := fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", path, err)
	}
	defer f.Close()

	var raw map[string]any
	if err := yaml.NewDecoder(f).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': While parsing config: %w", path, err)
	}

	return raw, nil
}

// applyRawConfig decodes the raw YAML map into cfg, preserving any existing
// default values for keys absent in the raw map.
func applyRawConfig(raw map[string]any, cfg *UncorsConfig) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           cfg,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToSliceHookFunc(","),
			StringToTimeDurationHookFunc(),
			URLMappingHookFunc(),
		),
	})
	if err != nil {
		return err
	}

	return decoder.Decode(raw)
}

// applyFlagOverrides applies CLI flag values to cfg, overriding any config file values.
// Only flags explicitly set on the command line are applied.
func applyFlagOverrides(cfg *UncorsConfig, flags *pflag.FlagSet) error {
	if flags.Changed("proxy") {
		cfg.Proxy, _ = flags.GetString("proxy")
	}

	if flags.Changed("debug") {
		cfg.Debug, _ = flags.GetBool("debug")
	}

	if flags.Changed("interactive") {
		cfg.Interactive, _ = flags.GetBool("interactive")
	}

	from, _ := flags.GetStringSlice("from")
	to, _ := flags.GetStringSlice("to")

	return mergeURLMappings(cfg, from, to)
}
