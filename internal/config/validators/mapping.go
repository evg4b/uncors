package validators

import (
	"fmt"

	"github.com/evg4b/uncors/internal/config"
	v "github.com/gobuffalo/validate"
)

type MappingValidator struct {
	Field string
	Value config.Mapping
}

func (m *MappingValidator) IsValid(errors *v.Errors) {
	errors.Append(v.Validate(
		&HostValidator{Field: "from", Value: m.Value.From},
		&HostValidator{Field: "to", Value: m.Value.To},
	))

	for i, static := range m.Value.Statics {
		errors.Append(v.Validate(&StaticValidator{
			Field: fmt.Sprintf("statics[%d]", i),
			Value: static,
		}))
	}

	for i, mock := range m.Value.Mocks {
		errors.Append(v.Validate(&MockValidator{
			Field: fmt.Sprintf("mocks[%d]", i),
			Value: mock,
		}))
	}

	for i, cache := range m.Value.Cache {
		errors.Append(v.Validate(&CacheValidator{
			Field: fmt.Sprintf("cache[%d]", i),
			Value: cache,
		}))
	}
}
