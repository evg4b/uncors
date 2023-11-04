package validators

import (
	"path"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators/base"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/gobuffalo/validate"
	"github.com/spf13/afero"
)

type StaticValidator struct {
	Field string
	Value config.StaticDirectory
	Fs    afero.Fs
}

func (s *StaticValidator) IsValid(errors *validate.Errors) {
	helpers.PassedOrOsFs(&s.Fs)

	errors.Append(validate.Validate(
		&base.PathValidator{
			Field: joinPath(s.Field, "path"),
			Value: s.Value.Path,
		},
		&base.DirectoryValidator{
			Field: joinPath(s.Field, "directory"),
			Value: s.Value.Dir,
			Fs:    s.Fs,
		},
	))

	if s.Value.Index != "" {
		errors.Append(validate.Validate(&base.FileValidator{
			Field: joinPath(s.Field, "index"),
			Value: path.Join(s.Value.Dir, s.Value.Index),
			Fs:    s.Fs,
		}))
	}
}
