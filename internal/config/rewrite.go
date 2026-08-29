package config

import (
	"errors"
	"fmt"
	"regexp"
	"slices"

	"github.com/evg4b/uncors/internal/urlpattern"
)

type RewritingOption struct {
	From string          `yaml:"from"`
	To   string          `yaml:"to"`
	Host urlpattern.Host `yaml:"host"`
}

func (r RewritingOption) Clone() RewritingOption {
	return r
}

type RewriteOptions []RewritingOption

func (r RewriteOptions) Clone() RewriteOptions {
	return slices.Clone(r)
}

func (r RewritingOption) Validate(field string) error {
	var errs []error

	errs = append(errs, ValidatePath(joinPath(field, "from"), r.From, true))
	errs = append(errs, ValidatePath(joinPath(field, "to"), r.To, true))
	errs = append(errs, r.validateVariables(field))

	if r.Host != (urlpattern.Host{}) {
		errs = append(errs, ValidateHost(joinPath(field, "host"), r.Host))
	}

	return errors.Join(errs...)
}

// validateVariables reports a {name} in `to` that `from` never captures.
// Substitution is a plain replacement, so such a variable would be left in the
// outgoing path literally rather than failing.
func (r RewritingOption) validateVariables(field string) error {
	captured := make(map[string]bool)
	for _, name := range pathVariables(r.From) {
		captured[name] = true
	}

	var errs []error

	for _, name := range pathVariables(r.To) {
		if !captured[name] {
			errs = append(errs, &ValidationError{
				fmt.Sprintf("%s references {%s}, which %s does not capture",
					joinPath(field, "to"), name, joinPath(field, "from")),
			})
		}
	}

	return errors.Join(errs...)
}

// pathVariableRegexp matches the {name} path variables gorilla/mux captures.
var pathVariableRegexp = regexp.MustCompile(`\{([a-zA-Z][a-zA-Z0-9_]*)[^}]*\}`)

func pathVariables(path string) []string {
	matches := pathVariableRegexp.FindAllStringSubmatch(path, -1)

	names := make([]string, 0, len(matches))
	for _, match := range matches {
		names = append(names, match[1])
	}

	return names
}
