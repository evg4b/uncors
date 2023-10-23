package validators

import (
	"github.com/evg4b/uncors/pkg/urlx"
	v "github.com/gobuffalo/validate"
)

type HostValidator struct {
	Field string
	Value string
}

func (h *HostValidator) IsValid(errors *v.Errors) {
	if !h.validateLength(errors) {
		return
	}

	parsed, err := urlx.Parse(h.Value)
	if err != nil {
		errors.Add(h.Field, "Host is invalid")

		return
	}

	if parsed.Hostname() == "" {
		errors.Add(h.Field, "Host is invalid")
	}

	if parsed.Port() != "" {
		errors.Add(h.Field, "Host must not contain port")
	}

	if parsed.Scheme != "" {
		errors.Add(h.Field, "Host must not contain scheme")
	}

	if parsed.User != nil {
		errors.Add(h.Field, "Host must not contain user")
	}

	if parsed.RawPath != "" {
		errors.Add(h.Field, "Host must not contain path")
	}

	if parsed.RawQuery != "" {
		errors.Add(h.Field, "Host must not contain query")
	}

	if parsed.RawFragment != "" {
		errors.Add(h.Field, "Host must not contain fragment")
	}

	if parsed.Opaque != "" {
		errors.Add(h.Field, "Host must not contain opaque")
	}

	if parsed.ForceQuery {
		errors.Add(h.Field, "Host must not contain force query")
	}
}

func (h *HostValidator) validateLength(errors *v.Errors) bool {
	if h.Value == "" {
		errors.Add(h.Field, "Host must not be empty")

		return false
	}

	if len(h.Value) > 255 {
		errors.Add(h.Field, "Host must be less than 255 characters")

		return false
	}

	return true
}
