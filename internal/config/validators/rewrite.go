package validators

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators/base"
	"github.com/gobuffalo/validate"
)

type RewritingOptionValidator struct {
	Field string
	Value config.RewritingOption
}

func (m *RewritingOptionValidator) IsValid(errors *validate.Errors) {
	errors.Append(validate.Validate(
		&base.PathValidator{
			Field: joinPath(m.Field, "from"),
			Value: m.Value.From,
		},
		&base.MethodValidator{
			Field:      joinPath(m.Field, "to"),
			Value:      m.Value.To,
			AllowEmpty: true,
		},
		&base.HostValidator{
			Field: joinPath(m.Field, "host"),
			Value: m.Value.Host,
		},
	))
}
