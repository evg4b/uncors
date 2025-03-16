package config

type Options struct {
	Disabled bool              `mapstructure:"disabled"`
	Headers  map[string]string `mapstructure:"headers"`
	Code     uint              `mapstructure:"code"`
}
