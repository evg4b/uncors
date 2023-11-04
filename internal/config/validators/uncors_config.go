package validators

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators/base"
	"github.com/gobuffalo/validate"
	"github.com/spf13/afero"
)

type UncorsConfigValidator struct {
	config *config.UncorsConfig
	Fs     afero.Fs
}

func (u *UncorsConfigValidator) IsValid(errors *validate.Errors) {
	errors.Append(validate.Validate(
		&base.PortValidator{Field: "http-port", Value: u.config.HTTPPort},
		&base.PortValidator{Field: "https-port", Value: u.config.HTTPSPort},
	))

	for i, mapping := range u.config.Mappings {
		errors.Append(validate.Validate(&MappingValidator{
			Field: joinPath("mappings", index(i)),
			Value: mapping,
		}))
	}

	errors.Append(validate.Validate(
		&ProxyValidator{Field: "proxy", Value: u.config.Proxy},
		&base.FileValidator{Field: "cert-file", Value: u.config.CertFile, Fs: u.Fs},
		&base.FileValidator{Field: "key-file", Value: u.config.KeyFile, Fs: u.Fs},
		&CacheConfigValidator{Field: "cache-config", Value: u.config.CacheConfig},
	))
}

func ValidateConfig(config *config.UncorsConfig, fs afero.Fs) error {
	errors := validate.Validate(&UncorsConfigValidator{
		config: config,
		Fs:     fs,
	})

	if errors.HasAny() {
		return errors
	}

	return nil
}
