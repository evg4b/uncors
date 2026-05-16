package config

import "github.com/evg4b/uncors/internal/helpers"

type OptionsHandling struct {
	Disabled bool              `yaml:"disabled"`
	Headers  map[string]string `yaml:"headers"`
	Code     int               `yaml:"code"`
}

func (o *OptionsHandling) Clone() OptionsHandling {
	return OptionsHandling{
		Disabled: o.Disabled,
		Headers:  helpers.CloneMap(o.Headers),
		Code:     o.Code,
	}
}

func (o *OptionsHandling) Validate(field string) error {
	if o.Code != 0 {
		return ValidateStatus(joinPath(field, "code"), o.Code)
	}

	return nil
}
