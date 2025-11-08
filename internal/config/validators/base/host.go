package base

import (
	"fmt"

	"github.com/evg4b/uncors/internal/urlparser"
	"github.com/gobuffalo/validate"
)

const maxHostLength = 255

type HostValidator struct {
	Field string
	Value string
}

func (h *HostValidator) IsValid(errors *validate.Errors) {
	if !h.validateHostLength(errors) {
		return
	}

	uri, err := urlparser.Parse(h.Value)
	if err != nil {
		errors.Add(h.Field, fmt.Sprintf("%s is not a valid host", h.Field))

		return
	}

	if uri.Path != "" {
		errors.Add(h.Field, fmt.Sprintf("%s must not contain a path", h.Field))
	}

	if uri.RawQuery != "" {
		errors.Add(h.Field, fmt.Sprintf("%s must not contain a query", h.Field))
	}

	if uri.Scheme != "http" && uri.Scheme != "https" && uri.Scheme != "" {
		errors.Add(h.Field, fmt.Sprintf("%s scheme must be http or https", h.Field))
	}
}

func (h *HostValidator) validateHostLength(errors *validate.Errors) bool {
	result := true
	length := len(h.Value)

	if length == 0 {
		errors.Add(h.Field, fmt.Sprintf("%s must not be empty", h.Field))

		result = false
	}

	if length > maxHostLength {
		errors.Add(h.Field, fmt.Sprintf("%s must not be longer than 255 characters, but got %d", h.Field, len(h.Value)))

		result = false
	}

	return result
}
