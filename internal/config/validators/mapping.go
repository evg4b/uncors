package validators

import (
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
			Field: joinPath(m.Field, "statics", index(i)),
			Value: static,
		}))
	}

	for i, mock := range m.Value.Mocks {
		errors.Append(validate.Validate(&MockValidator{
			Field: joinPath(m.Field, "mocks", index(i)),
			Value: mock,
		}))
	}

	for i, cache := range m.Value.Cache {
		errors.Append(validate.Validate(&CacheValidator{
			Field: joinPath(m.Field, cache, index(i)),
			Value: cache,
		}))
	}
}
