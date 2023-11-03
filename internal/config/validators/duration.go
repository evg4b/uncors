package validators

import (
	"fmt"
	"time"

	"github.com/gobuffalo/validate"
)

type DurationValidator struct {
	Field string
	Value time.Duration
}

func (d *DurationValidator) IsValid(errors *validate.Errors) {
	if d.Value <= 0 {
		errors.Add(d.Field, fmt.Sprintf("%s must be greater than 0", d.Field))
	}
}
