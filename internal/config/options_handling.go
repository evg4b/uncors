package config

type OptionsHandling struct {
	Disabled bool              `yaml:"disabled"`
	Headers  map[string]string `yaml:"headers"`
	Code     int               `yaml:"code"`
}

func (o *OptionsHandling) Validate(field string) error {
	if o.Code != 0 {
		return ValidateStatus(joinPath(field, "code"), o.Code)
	}

	return nil
}
