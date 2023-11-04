package validators

import (
	"fmt"
	"os"

	"github.com/evg4b/uncors/internal/helpers"

	"github.com/gobuffalo/validate"
	"github.com/spf13/afero"
)

type FileExistsValidator struct {
	Field string
	Value string
	Fs    afero.Fs
}

func (f *FileExistsValidator) IsValid(errors *validate.Errors) {
	helpers.PassedOrOsFs(&f.Fs)

	stat, err := f.Fs.Stat(f.Value)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			errors.Add(f.Field, fmt.Sprintf("%s file does not exist", f.Field))
		case os.IsPermission(err):
			errors.Add(f.Field, fmt.Sprintf("%s file is not accessible", f.Field))
		default:
			errors.Add(f.Field, fmt.Sprintf("%s is not a file", f.Field))
		}

		return
	}

	if stat.IsDir() {
		errors.Add(f.Field, fmt.Sprintf("%s is a directory", f.Field))
	}
}
