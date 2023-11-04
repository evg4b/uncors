package validators

import (
	"fmt"

	"github.com/gobuffalo/validate"
)

type StatusValidator struct {
	Field string
	Value int
}

func (s *StatusValidator) IsValid(errors *validate.Errors) {
	if s.Value < 100 || s.Value > 599 {
		errors.Add(s.Field, fmt.Sprintf("%s code must be in range 100-599", s.Field))
	}
}
