package validators

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators/base"
	"github.com/gobuffalo/validate"
)

type OptionsValidator struct {
	Field string
	Value config.Options
}

func (o *OptionsValidator) IsValid(errors *validate.Errors) {
	if o.Value.Code != 0 {
		errors.Append(validate.Validate(
			&base.StatusValidator{
				Field: joinPath(o.Field, "code"),
				Value: o.Value.Code,
			},
		))
	}
}
