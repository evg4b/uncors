package validators

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/gobuffalo/validate"
)

type StaticValidator struct {
	Field string
	Value config.StaticDirectory
}

func (s *StaticValidator) IsValid(errors *validate.Errors) {
	errors.Append(validate.Validate(&PathValidator{
		Field: joinPath(s.Field, "path"),
		Value: s.Value.Path,
	}))
}
