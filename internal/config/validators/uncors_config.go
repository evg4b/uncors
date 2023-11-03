package validators

import (
	"fmt"

	"github.com/evg4b/uncors/internal/config"
	"github.com/gobuffalo/validate"
)

type UncorsConfigValidator struct {
	config.UncorsConfig
}

func (u *UncorsConfigValidator) IsValid(errors *validate.Errors) {
	errors.Append(validate.Validate(
		&PortValidator{Field: "http-port", Value: u.HTTPPort},
		&PortValidator{Field: "https-port", Value: u.HTTPSPort},
	))

	for i, mapping := range u.Mappings {
		errors.Append(validate.Validate(&MappingValidator{
			Field: fmt.Sprintf("mappings[%d]", i),
			Value: mapping,
		}))
	}

	errors.Append(validate.Validate(
		&ProxyValidator{Field: "proxy", Value: u.Proxy},
		&FileExistsValidator{Field: "cert-file", Value: u.CertFile},
		&FileExistsValidator{Field: "key-file", Value: u.KeyFile},
		&CacheConfigValidator{Field: "cache-config", Value: u.CacheConfig},
	))
}

func ValidateConfig(config *config.UncorsConfig) error {
	if errors := validate.Validate(&UncorsConfigValidator{*config}); errors.HasAny() {
		return errors
	}

	return nil
}
