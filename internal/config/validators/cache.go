package validators

import "github.com/gobuffalo/validate"

type CacheValidator struct {
	Field string
	Value string
}

func (c *CacheValidator) IsValid(_ *validate.Errors) {
	// will be implemented later
}
