package validators

import "github.com/gobuffalo/validate"

type CacheValidator struct {
	Field string
	Value string
}

func (c *CacheValidator) IsValid(errors *validate.Errors) {

}
