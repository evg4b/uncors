package validators

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators/base"
	"github.com/gobuffalo/validate"
	"github.com/spf13/afero"
)

type ResponseValidator struct {
	Field string
	Value config.Response
	Fs    afero.Fs
}

func (r *ResponseValidator) IsValid(errors *validate.Errors) {
	errors.Append(validate.Validate(
		&base.StatusValidator{
			Field: joinPath(r.Field, "code"),
			Value: r.Value.Code,
		},
		&base.FileExistsValidator{
			Field: joinPath(r.Field, "file"),
			Value: r.Value.File,
			Fs:    r.Fs,
		},
		&base.DurationValidator{
			Field: joinPath(r.Field, "delay"),
			Value: r.Value.Delay,
		},
	))
}
