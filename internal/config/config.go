package config

import (
	"errors"
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

type UncorsConfig struct {
	Mappings    Mappings    `yaml:"mappings"`
	Proxy       string      `yaml:"proxy"`
	Debug       bool        `yaml:"debug"`
	CacheConfig CacheConfig `yaml:"cache-config"`
	Interactive bool        `yaml:"-"`
}

// LoadConfiguration reads the configuration file the flags point at and applies
// the flag overrides on top of it. Parsing the command line is the caller's job,
// so that a reload re-reads only the file.
func LoadConfiguration(fs afero.Fs, flags *Flags) (*UncorsConfig, error) {
	cfg := defaultConfig()

	configPath := flags.ConfigPath()
	if configPath != "" {
		err := readYAMLFile(fs, cfg, configPath)
		if err != nil {
			return nil, err
		}
	}

	err := applyFlagOverrides(cfg, flags.set)
	if err != nil {
		return nil, err
	}

	cfg.Mappings = NormaliseMappings(cfg.Mappings)

	err = cfg.Validate(fs)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

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

func (cfg *UncorsConfig) Validate(fs afero.Fs) error {
	if len(cfg.Mappings) == 0 {
		return &ValidationError{"mappings must not be empty"}
	}

	var errs []error

	for i, mapping := range cfg.Mappings {
		errs = append(errs, mapping.Validate(joinPath("mappings", index(i)), fs))
	}

	errs = append(errs, ValidateProxy("proxy", cfg.Proxy))
	errs = append(errs, cfg.CacheConfig.Validate("cache-config"))

	return errors.Join(errs...)
}
