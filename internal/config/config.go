package config

import (
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

// UncorsConfig is the root configuration for the uncors proxy.
type UncorsConfig struct {
	Mappings    Mappings    `yaml:"mappings"`
	Proxy       string      `yaml:"proxy"`
	Debug       bool        `yaml:"debug"`
	CacheConfig CacheConfig `yaml:"cache-config"`
	Interactive bool        `yaml:"-"`
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
		err := readYAMLFile(fs, cfg, configPath)
		if err != nil {
			return nil, "", err
		}
	}

	err = applyFlagOverrides(cfg, flags)
	if err != nil {
		return nil, "", err
	}

	cfg.Mappings = NormaliseMappings(cfg.Mappings)

	if err := cfg.Validate(fs); err != nil {
		return nil, "", err
	}

	return cfg, configPath, nil
}

// readYAMLFile opens a YAML config file and decodes it directly into cfg,
// preserving any existing default values for keys absent in the file.
func readYAMLFile(fs afero.Fs, cfg *UncorsConfig, path string) error {
	file, err := fs.Open(path)
	if err != nil {
		return fmt.Errorf("failed to read config file '%s': %w", path, err)
	}

	defer file.Close()

	err = yaml.NewDecoder(file).Decode(cfg)
	if err != nil {
		return fmt.Errorf("failed to read config file '%s': While parsing config: %w", path, err)
	}

	return nil
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

// Validate validates the full uncors configuration and returns a combined
// error listing all validation failures. Returns nil if the config is valid.
func (cfg *UncorsConfig) Validate(fs afero.Fs) error {
	var errs Errors

	if len(cfg.Mappings) == 0 {
		errs.add("mappings must not be empty")

		return errs
	}

	for i, mapping := range cfg.Mappings {
		mapping.Validate(joinPath("mappings", index(i)), fs, &errs)
	}

	ValidateProxy("proxy", cfg.Proxy, &errs)
	cfg.CacheConfig.Validate("cache-config", &errs)

	if errs.HasAny() {
		return errs
	}

	return nil
}
