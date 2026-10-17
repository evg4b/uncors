package config

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

type AllowedOrigins []string

func (o AllowedOrigins) Clone() AllowedOrigins {
	return slices.Clone(o)
}

func (o AllowedOrigins) Allows(origin string) bool {
	if len(o) == 0 || origin == "" {
		return true
	}

	for _, pattern := range o {
		if matched, _ := doublestar.Match(pattern, origin); matched {
			return true
		}
	}

	return false
}

func (o AllowedOrigins) Validate(field string) error {
	errs := make([]error, 0, len(o))

	for i, pattern := range o {
		errs = append(errs, validateOriginPattern(joinPath(field, index(i)), pattern))
	}

	return errors.Join(errs...)
}

func validateOriginPattern(field, pattern string) error {
	switch {
	case pattern == "":
		return &ValidationError{fmt.Sprintf("%s must not be empty", field)}
	case !doublestar.ValidatePattern(pattern):
		return &ValidationError{fmt.Sprintf("%s is not a valid glob pattern", field)}
	case strings.HasSuffix(pattern, "/"):
		return &ValidationError{fmt.Sprintf("%s must not end with a slash: an origin has no path", field)}
	}

	return nil
}
