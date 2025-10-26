package validators

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators/base"
	"github.com/gobuffalo/validate"
	"github.com/spf13/afero"
)

type UncorsConfigValidator struct {
	config *config.UncorsConfig
	fs     afero.Fs
}

func (u *UncorsConfigValidator) IsValid(errors *validate.Errors) {
	errors.Append(validate.Validate(
		&base.PortValidator{Field: "http-port", Value: u.config.HTTPPort},
	))

	if len(u.config.Mappings) == 0 {
		errors.Add("mappings", "mappings must not be empty")

		return
	}

	for i, mapping := range u.config.Mappings {
		errors.Append(validate.Validate(&MappingValidator{
			Field: joinPath("mappings", index(i)),
			Value: mapping,
			Fs:    u.fs,
		}))
	}

	errors.Append(validate.Validate(
		&ProxyValidator{Field: "proxy", Value: u.config.Proxy},
		&CacheConfigValidator{Field: "cache-config", Value: u.config.CacheConfig},
	))
}

func ValidateConfig(config *config.UncorsConfig, fs afero.Fs) error {
	errors := validate.Validate(&UncorsConfigValidator{
		config: config,
		fs:     fs,
	})

	if errors.HasAny() {
		return errors
	}

	return nil
}
