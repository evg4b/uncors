package config

import (
	"fmt"

	"github.com/evg4b/uncors/internal/middlewares/mock"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	defaultHTTPPort  = 80
	defaultHTTPSPort = 443
)

type UncorsConfig struct {
	// Base configuration
	HTTPPort int               `mapstructure:"http-port"`
	Mappings map[string]string `mapstructure:"mappings"`
	Proxy    string            `mapstructure:"proxy"`
	Debug    bool              `mapstructure:"debug"`
	// HTTPS configuration
	HTTPSPort int    `mapstructure:"https-port"`
	CertFile  string `mapstructure:"cert-file"`
	KeyFile   string `mapstructure:"key-file"`
	// Mocks configuration
	Mocks []mock.Mock
}

func (config UncorsConfig) IsHTTPSEnabled() bool {
	return len(config.CertFile) > 0 && len(config.KeyFile) > 0 && config.HTTPSPort > 0
}

func LoadConfiguration(viperInstance *viper.Viper) (*UncorsConfig, error) {
	defineFlags()
	pflag.Parse()
	if err := viperInstance.BindPFlags(pflag.CommandLine); err != nil {
		return nil, fmt.Errorf("filed parsing flags: %w", err)
	}

	configuration := &UncorsConfig{}
	if err := viperInstance.Unmarshal(configuration); err != nil {
		return nil, fmt.Errorf("filed parsing configuraion: %w", err)
	}

	if err := readURLMapping(viperInstance, configuration); err != nil {
		return nil, fmt.Errorf("recognize url mapping: %w", err)
	}

	return configuration, nil
}

func defineFlags() {
	pflag.StringSlice("to", []string{}, "Target host with protocol for to the resource to be proxy")
	pflag.StringSlice("from", []string{}, "Local host with protocol for to the resource from which proxying will take place") //nolint: lll
	pflag.Uint("http-port", defaultHTTPPort, "Local HTTP listening port")
	pflag.Uint("https-port", defaultHTTPSPort, "Local HTTPS listening port")
	pflag.String("cert-file", "", "Path to HTTPS certificate file")
	pflag.String("key-file", "", "Path to matching for certificate private key")
	pflag.String("proxy", "", "HTTP/HTTPS proxy to provide requests to real server (used system by default)")
	pflag.String("mocks", "", "File with configured mocks")
	pflag.Bool("debug", false, "Show debug output")
}
