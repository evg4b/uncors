package config

import "github.com/evg4b/uncors/internal/helpers"

type OptionsHandling struct {
	Disabled bool              `mapstructure:"disabled"`
	Headers  map[string]string `mapstructure:"headers"`
	Code     int               `mapstructure:"code"`
}

func (o *OptionsHandling) Clone() OptionsHandling {
	return OptionsHandling{
		Disabled: o.Disabled,
		Headers:  helpers.CloneMap(o.Headers),
		Code:     o.Code,
	}
}
