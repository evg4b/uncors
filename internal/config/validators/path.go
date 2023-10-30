package validators

import (
	"fmt"
	"strings"

	"github.com/evg4b/uncors/pkg/urlx"
	"github.com/gobuffalo/validate"
)

type PathValidator struct {
	Field string
	Value string
}

func (p *PathValidator) IsValid(errors *validate.Errors) {
	if p.Value == "" {
		errors.Add(p.Field, fmt.Sprintf("%s must not be empty", p.Field))

		return
	}

	if !strings.HasPrefix(p.Value, "/") {
		errors.Add(p.Field, fmt.Sprintf("%s must start with /", p.Field))

		return
	}

	uri, err := urlx.Parse("localhost" + p.Value)
	if err != nil {
		errors.Add(p.Field, fmt.Sprintf("%s is not valid path", p.Field))
	}

	if uri.RawQuery != "" {
		errors.Add(p.Field, fmt.Sprintf("%s must not contain query", p.Field))
	}
}
