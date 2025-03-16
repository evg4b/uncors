package config

import "github.com/evg4b/uncors/internal/helpers"

type Options struct {
	Disabled bool              `mapstructure:"disabled"`
	Headers  map[string]string `mapstructure:"headers"`
	Code     int               `mapstructure:"code"`
}

func (o *Options) Clone() Options {
	return Options{
		Disabled: o.Disabled,
		Headers:  helpers.CloneMap(o.Headers),
		Code:     o.Code,
	}
}

func (o *Options) String() string {
	return helpers.Sprintf("[code: %d]", o.Code)
}
