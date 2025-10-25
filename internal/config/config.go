package config

import (
	"fmt"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var flags *pflag.FlagSet

type UncorsConfig struct {
	HTTPPort    int         `mapstructure:"http-port"`
	Mappings    Mappings    `mapstructure:"mappings"`
	Proxy       string      `mapstructure:"proxy"`
	Debug       bool        `mapstructure:"debug"`
	HTTPSPort   int         `mapstructure:"https-port"`
	CertFile    string      `mapstructure:"cert-file"`
	KeyFile     string      `mapstructure:"key-file"`
	CacheConfig CacheConfig `mapstructure:"cache-config"`
}

func (c *UncorsConfig) IsHTTPSEnabled() bool {
	return len(c.CertFile) > 0 && len(c.KeyFile) > 0
}

func LoadConfiguration(viperInstance *viper.Viper, args []string) *UncorsConfig {
	defineFlags()
	helpers.AssertIsDefined(flags)
	if err := flags.Parse(args); err != nil {
		panic(fmt.Errorf("failed parsing flags: %w", err))
	}

	if err := viperInstance.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("failed binding flags: %w", err))
	}

	configuration := &UncorsConfig{
		Mappings: []Mapping{},
	}

	if configPath := viperInstance.GetString("config"); len(configPath) > 0 {
		viperInstance.SetConfigFile(configPath)
		if err := viperInstance.ReadInConfig(); err != nil {
			panic(fmt.Errorf("failed to read config file '%s': %w", configPath, err))
		}
	}

	configOption := viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToSliceHookFunc(","),
		StringToTimeDurationHookFunc(),
		URLMappingHookFunc(),
	))

	setDefaultValues(viperInstance)
	if err := viperInstance.Unmarshal(configuration, configOption); err != nil {
		panic(fmt.Errorf("failed parsing config: %w", err))
	}

	if err := readURLMapping(viperInstance, configuration); err != nil {
		panic(err)
	}

	configuration.Mappings = NormaliseMappings(configuration.Mappings)

	return configuration
}

func defineFlags() {
	flags = pflag.NewFlagSet("uncors", pflag.ContinueOnError)
	flags.Usage = pflag.Usage
	flags.StringSliceP("to", "t", []string{}, "Target host with protocol for the resource to be proxied")
	flags.StringSliceP("from", "f", []string{}, "Local host with protocol for the resource from which proxying will take place") //nolint: lll
	flags.UintP("http-port", "p", defaultHTTPPort, "Local HTTP listening port")
	flags.UintP("https-port", "s", 0, "Local HTTPS listening port")
	flags.String("cert-file", "", "Path to HTTPS certificate file")
	flags.String("key-file", "", "Path to private key file matching the certificate")
	flags.String("proxy", "", "HTTP/HTTPS proxy for requests to the real server (uses system proxy by default)")
	flags.Bool("debug", false, "Show debug output")
	flags.StringP("config", "c", "", "Path to the configuration file")
}
