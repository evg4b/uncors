package validators

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators/base"
	"github.com/gobuffalo/validate"
	"github.com/spf13/afero"
)

type MappingValidator struct {
	Field string
	Value config.Mapping
	Fs    afero.Fs
}

func (m *MappingValidator) IsValid(errors *validate.Errors) {
	errors.Append(validate.Validate(
		&base.HostValidator{Field: joinPath(m.Field, "from"), Value: m.Value.From},
		&base.HostValidator{Field: joinPath(m.Field, "to"), Value: m.Value.To},
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
			Fs:    m.Fs,
		}))
	}

	for i, cache := range m.Value.Cache {
		errors.Append(validate.Validate(&CacheValidator{
			Field: joinPath(m.Field, "cache", index(i)),
			Value: cache,
		}))
	}

	for i, rewrite := range m.Value.Rewrite {
		errors.Append(validate.Validate(&RewritingOptionValidator{
			Field: joinPath(m.Field, "rewrite", index(i)),
			Value: rewrite,
		}))
	}
}
