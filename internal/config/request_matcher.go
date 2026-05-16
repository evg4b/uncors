package config

import (
	multierror "github.com/hashicorp/go-multierror"

	"github.com/evg4b/uncors/internal/helpers"
)

type RequestMatcher struct {
	Path    string            `yaml:"path"`
	Method  string            `yaml:"method"`
	Queries map[string]string `yaml:"queries"`
	Headers map[string]string `yaml:"headers"`
}

func (r *RequestMatcher) Clone() RequestMatcher {
	return RequestMatcher{
		Path:    r.Path,
		Method:  r.Method,
		Queries: helpers.CloneMap(r.Queries),
		Headers: helpers.CloneMap(r.Headers),
	}
}

func (r *RequestMatcher) IsPathOnly() bool {
	return r.Method == "" && len(r.Queries) == 0 && len(r.Headers) == 0
}

func (r *RequestMatcher) Validate(field string) error {
	var errs *multierror.Error

	errs = multierror.Append(errs, ValidatePath(joinPath(field, "path"), r.Path, false))
	errs = multierror.Append(errs, ValidateMethod(joinPath(field, "method"), r.Method, true))

	return joinErrors(errs)
}
