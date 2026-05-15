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
		readErr := readYAMLFile(fs, cfg, configPath)
		if readErr != nil {
			return nil, "", readErr
		}
	}

	err = applyFlagOverrides(cfg, flags)
	if err != nil {
		return nil, "", err
	}

	cfg.Mappings = NormaliseMappings(cfg.Mappings)

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
