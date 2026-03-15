package validators

import (
	"fmt"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators/base"
	"github.com/gobuffalo/validate"
)

type CacheConfigValidator struct {
	Field string
	Value config.CacheConfig
}

func (c *CacheConfigValidator) IsValid(errors *validate.Errors) {
	errors.Append(validate.Validate(
		&base.DurationValidator{
			Field: joinPath(c.Field, "expiration-time"),
			Value: c.Value.ExpirationTime,
		},
	))

	if c.Value.MaxSize <= 0 {
		maxSizeField := joinPath(c.Field, "max-size")
		errors.Add(maxSizeField, fmt.Sprintf("%s must be greater than 0", maxSizeField))
	}

	if len(c.Value.Methods) == 0 {
		errors.Add(c.Field, "methods must not be empty")
	}

	for i, method := range c.Value.Methods {
		errors.Append(validate.Validate(
			&base.MethodValidator{
				Field: joinPath(c.Field, "methods", index(i)),
				Value: method,
			},
		))
	}
}
