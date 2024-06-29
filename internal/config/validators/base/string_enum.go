package base

import (
	"fmt"

	"github.com/gobuffalo/validate"
)

type StringEnumValidator struct {
	Field   string
	Value   string
	Options []string
}

func (f *StringEnumValidator) IsValid(errors *validate.Errors) {
	for _, option := range f.Options {
		if f.Value == option {
			return
		}
	}

	errors.Add(f.Field, fmt.Sprintf("'%s' is not a valid option", f.Value))
}
