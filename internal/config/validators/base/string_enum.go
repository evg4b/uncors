package base

import (
	"fmt"
	"slices"

	"github.com/gobuffalo/validate"
)

type StringEnumValidator struct {
	Field   string
	Value   string
	Options []string
}

func (f *StringEnumValidator) IsValid(errors *validate.Errors) {
	if slices.Contains(f.Options, f.Value) {
		return
	}

	errors.Add(f.Field, fmt.Sprintf("'%s' is not a valid option", f.Value))
}
