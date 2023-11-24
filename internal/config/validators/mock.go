package validators

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators/base"
	"github.com/gobuffalo/validate"
	"github.com/spf13/afero"
)

type MockValidator struct {
	Field string
	Value config.Mock
	Fs    afero.Fs
}

func (m *MockValidator) IsValid(errors *validate.Errors) {
	errors.Append(validate.Validate(
		&base.PathValidator{
			Field: joinPath(m.Field, "path"),
			Value: m.Value.Path,
		},
		&base.MethodValidator{
			Field:      joinPath(m.Field, "method"),
			Value:      m.Value.Method,
			AllowEmpty: true,
		},
		&ResponseValidator{
			Field: joinPath(m.Field, "response"),
			Value: m.Value.Response,
			Fs:    m.Fs,
		},
	))
}
