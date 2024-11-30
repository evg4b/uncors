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

	if r.validateFiles(errors) {
		return
	}

	if r.Value.File != "" {
		errors.Append(validate.Validate(&base.FileValidator{
			Field: joinPath(r.Field, "file"),
			Value: r.Value.File,
			Fs:    r.Fs,
		}))
	}

	if r.Value.Fake != nil {
		errors.Append(validate.Validate(&FakedataNodeValidator{
			Field: joinPath(r.Field, "fake"),
			Value: r.Value.Fake,
			Root:  true,
		}))
	}
}

func (r *ResponseValidator) validateFiles(errors *validate.Errors) bool {
	nodes := make([]string, 0, 3) //nolint:mnd

	if r.Value.Raw != "" {
		nodes = append(nodes, joinPath(r.Field, "raw"))
	}
	if r.Value.File != "" {
		nodes = append(nodes, joinPath(r.Field, "file"))
	}
	if r.Value.Fake != nil {
		nodes = append(nodes, joinPath(r.Field, "fake"))
	}

	switch len(nodes) {
	case 0:
		errors.Add(r.Field, fmt.Sprintf(
			"%s, %s or %s must be set",
			joinPath(r.Field, "raw"),
			joinPath(r.Field, "file"),
			joinPath(r.Field, "fake"),
		))
	case 1:
		return false
	case 2: //nolint:mnd
		errors.Add(r.Field, fmt.Sprintf("only one of %s or %s must be set", nodes[0], nodes[1]))
	case 3: //nolint:mnd
		errors.Add(r.Field, fmt.Sprintf("only one of %s, %s or %s must be set", nodes[0], nodes[1], nodes[2]))
	}

	return true
}
