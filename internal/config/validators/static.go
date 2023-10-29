package validators

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/gobuffalo/validate"
)

type StaticValidator struct {
	Field string
	Value config.StaticDirectory
}

func (s *StaticValidator) IsValid(_ *validate.Errors) {
	// will be implemented later
}
