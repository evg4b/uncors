package base

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gobuffalo/validate"
	"github.com/samber/lo"
)

type MethodValidator struct {
	Field      string
	Value      string
	AllowEmpty bool
}

var allowedMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodConnect,
	http.MethodOptions,
	http.MethodTrace,
}

var lastAllowedMethodIndex = len(allowedMethods) - 1

func (m *MethodValidator) IsValid(errors *validate.Errors) {
	if m.AllowEmpty && m.Value == "" {
		return
	}

	if !lo.Contains(allowedMethods, m.Value) {
		builder := strings.Builder{}
		builder.WriteString(fmt.Sprintf("%s must be one of ", m.Field))
		for i, allowedMethod := range allowedMethods {
			builder.WriteString(allowedMethod)
			if i != lastAllowedMethodIndex {
				builder.WriteString(", ")
			}
		}

		errors.Add(m.Field, builder.String())
	}
}
