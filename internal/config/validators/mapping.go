package validators

import (
	"fmt"

	"github.com/evg4b/uncors/internal/config"
	"github.com/gobuffalo/validate"
)

type MappingValidator struct {
	Field string
	Value config.Mapping
}

func (m *MappingValidator) IsValid(errors *validate.Errors) {
	errors.Append(validate.Validate(
		&HostValidator{Field: "from", Value: m.Value.From},
		&HostValidator{Field: "to", Value: m.Value.To},
	))

	for i, static := range m.Value.Statics {
		errors.Append(validate.Validate(&StaticValidator{
			Field: joinPath(m.Field, fmt.Sprintf("statics[%d]", i)),
			Value: static,
		}))
	}

	for i, mock := range m.Value.Mocks {
		errors.Append(validate.Validate(&MockValidator{
			Field: joinPath(m.Field, fmt.Sprintf("mocks[%d]", i)),
			Value: mock,
		}))
	}

	for i, cache := range m.Value.Cache {
		errors.Append(validate.Validate(&CacheValidator{
			Field: joinPath(m.Field, fmt.Sprintf("cache[%d]", i)),
			Value: cache,
		}))
	}
}
