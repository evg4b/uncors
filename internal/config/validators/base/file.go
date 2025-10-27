package base

import (
	"fmt"
	"os"

	"github.com/gobuffalo/validate"
	"github.com/spf13/afero"
)

type FileValidator struct {
	Field string
	Value string
	Fs    afero.Fs
}

func (f *FileValidator) IsValid(errors *validate.Errors) {
	stat, err := f.Fs.Stat(f.Value)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			errors.Add(f.Field, fmt.Sprintf("%s %s does not exist", f.Field, f.Value))
		case os.IsPermission(err):
			errors.Add(f.Field, fmt.Sprintf("%s %s is not accessible", f.Field, f.Value))
		default:
			errors.Add(f.Field, fmt.Sprintf("%s %s is not a file", f.Field, f.Value))
		}

		return
	}

	if stat.IsDir() {
		errors.Add(f.Field, fmt.Sprintf("%s %s is a directory", f.Field, f.Value))
	}
}
