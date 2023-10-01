package config

import (
	"github.com/go-playground/validator/v10"
)

func Validate(c *UncorsConfig) error {
	validate := validator.New()

	return validate.Struct(c)
}
