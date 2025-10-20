package validators

import (
	"fmt"

	"github.com/gobuffalo/validate"
	"github.com/spf13/afero"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators/base"
	"github.com/evg4b/uncors/internal/helpers"
)

type ScriptValidator struct {
	Field string
	Value config.Script
	Fs    afero.Fs
}

func (s *ScriptValidator) IsValid(errors *validate.Errors) {
	helpers.PassedOrOsFs(&s.Fs)

	errors.Append(validate.Validate(
		&base.PathValidator{
			Field: joinPath(s.Field, "path"),
			Value: s.Value.Path,
		},
	))

	errors.Append(validate.Validate(
		&base.MethodValidator{
			Field:      joinPath(s.Field, "method"),
			Value:      s.Value.Method,
			AllowEmpty: true,
		},
	))

	if s.Value.Script == "" && s.Value.File == "" {
		errors.Add(
			joinPath(s.Field, "script"),
			fmt.Sprintf("%s: either 'script' or 'file' must be provided", joinPath(s.Field, "script")),
		)
		errors.Add(
			joinPath(s.Field, "file"),
			fmt.Sprintf("%s: either 'script' or 'file' must be provided", joinPath(s.Field, "file")),
		)
	}

	if s.Value.Script != "" && s.Value.File != "" {
		errors.Add(
			joinPath(s.Field, "script"),
			fmt.Sprintf("%s: only one of 'script' or 'file' can be provided", joinPath(s.Field, "script")),
		)
		errors.Add(
			joinPath(s.Field, "file"),
			fmt.Sprintf("%s: only one of 'script' or 'file' can be provided", joinPath(s.Field, "file")),
		)
	}

	if s.Value.File != "" {
		errors.Append(validate.Validate(&base.FileValidator{
			Field: joinPath(s.Field, "file"),
			Value: s.Value.File,
			Fs:    s.Fs,
		}))
	}

	for key := range s.Value.Queries {
		if key == "" {
			errors.Add(
				joinPath(s.Field, "queries"),
				fmt.Sprintf("%s: query parameter keys must not be empty", joinPath(s.Field, "queries")),
			)
		}
	}

	for key := range s.Value.Headers {
		if key == "" {
			errors.Add(
				joinPath(s.Field, "headers"),
				fmt.Sprintf("%s: header keys must not be empty", joinPath(s.Field, "headers")),
			)
		}
	}
}
