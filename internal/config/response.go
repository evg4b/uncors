package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/spf13/afero"
)

type Response struct {
	Code    int               `yaml:"code"`
	Headers map[string]string `yaml:"headers"`
	Delay   time.Duration     `yaml:"delay"`
	Raw     string            `yaml:"raw"`
	File    string            `yaml:"file"`
}

func (r *Response) Clone() Response {
	return Response{
		Code:    r.Code,
		Headers: helpers.CloneMap(r.Headers),
		Raw:     r.Raw,
		File:    r.File,
		Delay:   r.Delay,
	}
}

func (r *Response) IsRaw() bool {
	return len(r.Raw) > 0
}

func (r *Response) IsFile() bool {
	return len(r.File) > 0
}

func (r *Response) Validate(field string, fs afero.Fs) error {
	var errs []error

	errs = append(errs, ValidateStatus(joinPath(field, "code"), r.Code))
	errs = append(errs, ValidateDuration(joinPath(field, "delay"), r.Delay, true))

	switch {
	case r.Raw == "" && r.File == "":
		errs = append(errs, &ValidationError{fmt.Sprintf(
			"%s or %s must be set",
			joinPath(field, "raw"),
			joinPath(field, "file"),
		)})
	case r.Raw != "" && r.File != "":
		errs = append(errs, &ValidationError{fmt.Sprintf(
			"only one of %s or %s must be set",
			joinPath(field, "raw"),
			joinPath(field, "file"),
		)})
	case r.File != "":
		errs = append(errs, ValidateFile(joinPath(field, "file"), r.File, fs))
	}

	return errors.Join(errs...)
}
