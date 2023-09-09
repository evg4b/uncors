package config

import (
	"github.com/go-playground/validator/v10"
)

func validate(c *UncorsConfig) error {
	validate := validator.New()

	return validate.Struct(c) //nolint:wrapcheck
}
