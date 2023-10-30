package validators

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/gobuffalo/validate"
)

type MockValidator struct {
	Field string
	Value config.Mock
}

func (m *MockValidator) IsValid(_ *validate.Errors) {
	// will be implemented later
}
