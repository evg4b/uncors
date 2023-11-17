package base

import (
	"fmt"

	"github.com/gobuffalo/validate"
)

type PortValidator struct {
	Field string
	Value int
}

func (p *PortValidator) IsValid(errors *validate.Errors) {
	if p.Value < 1 || p.Value > 65535 {
		errors.Add(p.Field, fmt.Sprintf("%s must be between 0 and 65535", p.Field))
	}
}
