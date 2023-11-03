package validators

import (
	"fmt"

	"github.com/evg4b/uncors/pkg/urlx"
	"github.com/gobuffalo/validate"
)

type ProxyValidator struct {
	Field string
	Value string
}

func (p *ProxyValidator) IsValid(errors *validate.Errors) {
	if p.Value == "" {
		return
	}

	if _, err := urlx.Parse(p.Value); err != nil {
		errors.Add(p.Field, fmt.Sprintf("%s is not valid url", p.Field))
	}
}
