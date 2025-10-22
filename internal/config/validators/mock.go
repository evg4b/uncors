package validators

import (
	"github.com/evg4b/uncors/internal/config"
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
		&RequestMatcherValidator{
			Field: m.Field,
			Value: m.Value.RequestMatcher,
		},
		&ResponseValidator{
			Field: joinPath(m.Field, "response"),
			Value: m.Value.Response,
			Fs:    m.Fs,
		},
	))
}
