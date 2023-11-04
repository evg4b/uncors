package validators

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/gobuffalo/validate"
)

type MockValidator struct {
	Field string
	Value config.Mock
}

func (m *MockValidator) IsValid(errors *validate.Errors) {
	errors.Append(validate.Validate(
		&PathValidator{
			Field: joinPath(m.Field, "path"),
			Value: m.Value.Path,
		},
		&MethodValidator{
			Field:      joinPath(m.Field, "method"),
			Value:      m.Value.Method,
			AllowEmpty: true,
		},
		&ResponseValidator{
			Field: joinPath(m.Field, "response"),
			Value: m.Value.Response,
		},
	))
}
