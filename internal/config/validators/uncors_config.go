package validators

import (
	"fmt"

	"github.com/evg4b/uncors/internal/config"
	"github.com/gobuffalo/validate"
)

type UncorsConfigValidator struct {
	Config *config.UncorsConfig
}

func (u *UncorsConfigValidator) IsValid(errors *validate.Errors) {
	errors.Append(validate.Validate(
		&PortValidator{Field: "http-port", Value: u.Config.HTTPPort},
		&PortValidator{Field: "https-port", Value: u.Config.HTTPSPort},
	))

	for i, mapping := range u.Config.Mappings {
		errors.Append(validate.Validate(&MappingValidator{
			Field: fmt.Sprintf("mappings[%d]", i),
			Value: mapping,
		}))
	}
}

func ValidateConfig(config *config.UncorsConfig) error {
	errors := validate.Validate(&UncorsConfigValidator{Config: config})
	if errors.HasAny() {
		return errors
	}

	return nil
}
