package config

import (
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

type UncorsConfig struct {
	Mappings Mappings `yaml:"mappings"`
	Proxy    string   `yaml:"proxy"`
	Debug    bool     `yaml:"debug"`
	// Listen is the address the proxy binds to. It defaults to loopback, because
	// uncors disables CORS protections and must not be reachable by default;
	// binding anything else is an explicit, warned-about choice.
	Listen      string      `yaml:"listen"`
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
	cfg.Proxy = NormaliseProxy(cfg.Proxy)
	resolvePaths(cfg, configPath)

	err = cfg.Validate(fs)
	if err != nil {
		return nil, err
	}

	warnAboutShadowedRoutes(cfg)

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

	warnAboutUnknownFields(fs, path)

	return nil
}

// warnAboutUnknownFields reports keys the configuration does not have. A
// misspelled or removed key is otherwise accepted in silence, and the user is
// left believing they enabled something they did not — which is worse than an
// error. It is a warning rather than a failure because upgrading configs are
// likely to carry keys from older versions.
func warnAboutUnknownFields(fs afero.Fs, path string) {
	file, err := fs.Open(path)
	if err != nil {
		return
	}

	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	err = decoder.Decode(&UncorsConfig{})
	if err != nil && !errors.Is(err, io.EOF) {
		slog.Warn("the configuration file contains unknown keys, which uncors ignores",
			"path", path, "detail", err)
	}
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

	if flags.Changed("listen") {
		cfg.Listen, _ = flags.GetString("listen")
	}

	if cfg.Listen == "" {
		cfg.Listen = DefaultListenAddress
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
	errs = append(errs, ValidateListenAddress("listen", cfg.Listen))
	errs = append(errs, cfg.CacheConfig.Validate("cache-config"))

	return errors.Join(errs...)
}
