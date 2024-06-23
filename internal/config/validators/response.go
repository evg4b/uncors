package validators

import (
	"fmt"

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
		&base.DurationValidator{
			Field:     joinPath(r.Field, "delay"),
			Value:     r.Value.Delay,
			AllowZero: true,
		},
	))

	if r.Value.Raw == "" && r.Value.File == "" && r.Value.Fake == nil {
		errors.Add(r.Field, fmt.Sprintf(
			"%s, %s or %s  must be set",
			joinPath(r.Field, "raw"),
			joinPath(r.Field, "file"),
			joinPath(r.Field, "fake"),
		))

		return
	}

	if r.Value.Raw != "" && r.Value.File != "" {
		rawPath := joinPath(r.Field, "raw")
		filePath := joinPath(r.Field, "file")
		errors.Add(r.Field, fmt.Sprintf("only one of %s or %s must be set", rawPath, filePath))

		return
	}

	if r.Value.File != "" {
		errors.Append(validate.Validate(&base.FileValidator{
			Field: joinPath(r.Field, "file"),
			Value: r.Value.File,
			Fs:    r.Fs,
		}))
	}
}
