package validators

import (
	"fmt"

	"github.com/gobuffalo/validate"
	"github.com/spf13/afero"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators/base"
	"github.com/evg4b/uncors/internal/helpers"
)

type LuaScriptValidator struct {
	Field string
	Value config.LuaScript
	Fs    afero.Fs
}

func (l *LuaScriptValidator) IsValid(errors *validate.Errors) {
	helpers.PassedOrOsFs(&l.Fs)

	// Validate path (required)
	errors.Append(validate.Validate(
		&base.PathValidator{
			Field: joinPath(l.Field, "path"),
			Value: l.Value.Path,
		},
	))

	// Validate method (optional, but must be valid if provided)
	errors.Append(validate.Validate(
		&base.MethodValidator{
			Field:      joinPath(l.Field, "method"),
			Value:      l.Value.Method,
			AllowEmpty: true,
		},
	))

	// Validate script/file mutual exclusivity
	if l.Value.Script == "" && l.Value.File == "" {
		errors.Add(
			joinPath(l.Field, "script"),
			fmt.Sprintf("%s: either 'script' or 'file' must be provided", joinPath(l.Field, "script")),
		)
		errors.Add(
			joinPath(l.Field, "file"),
			fmt.Sprintf("%s: either 'script' or 'file' must be provided", joinPath(l.Field, "file")),
		)
	}

	if l.Value.Script != "" && l.Value.File != "" {
		errors.Add(
			joinPath(l.Field, "script"),
			fmt.Sprintf("%s: only one of 'script' or 'file' can be provided", joinPath(l.Field, "script")),
		)
		errors.Add(
			joinPath(l.Field, "file"),
			fmt.Sprintf("%s: only one of 'script' or 'file' can be provided", joinPath(l.Field, "file")),
		)
	}

	// Validate file existence if file is specified
	if l.Value.File != "" {
		errors.Append(validate.Validate(&base.FileValidator{
			Field: joinPath(l.Field, "file"),
			Value: l.Value.File,
			Fs:    l.Fs,
		}))
	}

	// Validate queries - keys should not be empty
	for key := range l.Value.Queries {
		if key == "" {
			errors.Add(
				joinPath(l.Field, "queries"),
				fmt.Sprintf("%s: query parameter keys must not be empty", joinPath(l.Field, "queries")),
			)
		}
	}

	// Validate headers - keys should not be empty
	for key := range l.Value.Headers {
		if key == "" {
			errors.Add(
				joinPath(l.Field, "headers"),
				fmt.Sprintf("%s: header keys must not be empty", joinPath(l.Field, "headers")),
			)
		}
	}
}
