package config

import (
	"errors"
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

// ErrVersionRequested is returned when the --version flag is set so that the
// caller can exit cleanly after the version has been printed.
var ErrVersionRequested = errors.New("version requested")

type UncorsConfig struct {
	Mappings    Mappings    `yaml:"mappings"`
	Proxy       string      `yaml:"proxy"`
	CacheConfig CacheConfig `yaml:"cache-config"`
	Interactive bool        `yaml:"-"`
}

func LoadConfiguration(fs afero.Fs, version string, args []string) (*UncorsConfig, string, error) {
	flags, err := defineFlags(version)
	if err != nil {
		return nil, "", err
	}

	err = flags.Parse(args)
	if err != nil {
		return nil, "", fmt.Errorf("failed parsing flags: %w", err)
	}

	printVersion, err := flags.GetBool("version")
	if err != nil {
		return nil, "", err
	}
	if printVersion {
		return nil, "", ErrVersionRequested
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

	err = cfg.Validate(fs)
	if err != nil {
		return nil, "", err
	}

	return cfg, configPath, nil
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
