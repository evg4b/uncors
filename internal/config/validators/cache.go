package validators

import (
	"fmt"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/gobuffalo/validate"
)

type CacheValidator struct {
	Field string
	Value string
}

func (c *CacheValidator) IsValid(errors *validate.Errors) {
	if !doublestar.ValidatePathPattern(c.Value) {
		errors.Add(c.Field, fmt.Sprintf("%s is not a valid glob pattern", c.Field))
	}
}
