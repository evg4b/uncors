package base

import (
	"fmt"
	"os"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/gobuffalo/validate"
	"github.com/spf13/afero"
)

type DirectoryValidator struct {
	Field string
	Value string
	Fs    afero.Fs
}

func (f *DirectoryValidator) IsValid(errors *validate.Errors) {
	if f.Value == "" {
		errors.Add(f.Field, fmt.Sprintf("%s must not be empty", f.Field))

		return
	}

	helpers.PassedOrOsFs(&f.Fs)

	stat, err := f.Fs.Stat(f.Value)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			errors.Add(f.Field, fmt.Sprintf("%s directory does not exist", f.Field))
		case os.IsPermission(err):
			errors.Add(f.Field, fmt.Sprintf("%s directory is not accessible", f.Field))
		default:
			errors.Add(f.Field, fmt.Sprintf("%s is not a directory", f.Field))
		}

		return
	}

	if !stat.IsDir() {
		errors.Add(f.Field, fmt.Sprintf("%s is not a directory", f.Field))
	}
}
