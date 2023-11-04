package validators

import (
	"fmt"
	"os"

	"github.com/gobuffalo/validate"
	"github.com/spf13/afero"
)

type FileExistsValidator struct {
	Field string
	Value string
	Fs    afero.Fs
}

func (f *FileExistsValidator) IsValid(errors *validate.Errors) {
	if f.Fs == nil {
		f.Fs = afero.NewOsFs()
	}

	stat, err := f.Fs.Stat(f.Value)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			errors.Add(f.Field, fmt.Sprintf("%s does not exist", f.Value))
		case os.IsPermission(err):
			errors.Add(f.Field, fmt.Sprintf("%s is not accessible", f.Value))
		default:
			errors.Add(f.Field, fmt.Sprintf("%s is not a file", f.Value))
		}

		return
	}

	if stat.IsDir() {
		errors.Add(f.Field, fmt.Sprintf("%s is a directory", f.Value))
	}
}
