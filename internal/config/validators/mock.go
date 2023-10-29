package validators

import (
	"github.com/evg4b/uncors/internal/config"
	v "github.com/gobuffalo/validate"
)

type MockValidator struct {
	Field string
	Value config.Mock
}

func (m *MockValidator) IsValid(_ *v.Errors) {
	// will be implemented later
}
