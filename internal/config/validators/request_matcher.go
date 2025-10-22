package validators

import (
	"github.com/gobuffalo/validate"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators/base"
)

type RequestMatcherValidator struct {
	Field string
	Value config.RequestMatcher
}

func (r *RequestMatcherValidator) IsValid(errors *validate.Errors) {
	errors.Append(validate.Validate(
		&base.PathValidator{
			Field: joinPath(r.Field, "path"),
			Value: r.Value.Path,
		},
	))

	errors.Append(validate.Validate(
		&base.MethodValidator{
			Field:      joinPath(r.Field, "method"),
			Value:      r.Value.Method,
			AllowEmpty: true,
		},
	))
}
