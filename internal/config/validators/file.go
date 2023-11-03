package validators

import "github.com/gobuffalo/validate"

type FileExistsValidator struct {
	Field string
	Value string
}

func (f *FileExistsValidator) IsValid(_ *validate.Errors) {
	// TODO implement me
}
