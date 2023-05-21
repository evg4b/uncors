package config

import (
	"github.com/go-playground/validator/v10"
)

func Validate(config *UncorsConfig) error {
	validate := validator.New()

	return validate.Struct(config) //nolint:wrapcheck
}
