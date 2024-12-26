package base

import (
	"fmt"
	"strings"

	"github.com/evg4b/uncors/pkg/urlx"
	"github.com/gobuffalo/validate"
)

type PathValidator struct {
	Field    string
	Value    string
	Relative bool
}

func (p *PathValidator) IsValid(errors *validate.Errors) {
	if p.Value == "" {
		errors.Add(p.Field, fmt.Sprintf("%s must not be empty", p.Field))

		return
	}

	if !p.Relative && !strings.HasPrefix(p.Value, "/") {
		errors.Add(p.Field, fmt.Sprintf("%s must be absolute and start with /", p.Field))

		return
	}

	uri, err := urlx.Parse("//localhost/" + strings.TrimPrefix(p.Value, "/"))
	if err != nil {
		errors.Add(p.Field, fmt.Sprintf("%s is not valid path", p.Field))
	}

	if uri.RawQuery != "" {
		errors.Add(p.Field, fmt.Sprintf("%s must not contain query", p.Field))
	}
}
