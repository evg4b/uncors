package base

import (
	"fmt"
	"time"

	"github.com/gobuffalo/validate"
)

type DurationValidator struct {
	Field     string
	Value     time.Duration
	AllowZero bool
}

func (d *DurationValidator) IsValid(errors *validate.Errors) {
	if d.AllowZero {
		d.validateWithZero(errors)
	} else {
		d.validateWithoutZero(errors)
	}
}

func (d *DurationValidator) validateWithoutZero(errors *validate.Errors) {
	if d.Value <= 0 {
		errors.Add(d.Field, fmt.Sprintf("%s must be greater than 0", d.Field))
	}
}

func (d *DurationValidator) validateWithZero(errors *validate.Errors) {
	if d.Value < 0 {
		errors.Add(d.Field, fmt.Sprintf("%s must be greater than or equal to 0", d.Field))
	}
}
