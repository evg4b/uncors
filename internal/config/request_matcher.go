package config

import (
	"errors"
)

type RequestMatcher struct {
	Path    string            `yaml:"path"`
	Method  string            `yaml:"method"`
	Queries map[string]string `yaml:"queries"`
	Headers map[string]string `yaml:"headers"`
}

func (r *RequestMatcher) IsPathOnly() bool {
	return r.Method == "" && len(r.Queries) == 0 && len(r.Headers) == 0
}

func (r *RequestMatcher) Validate(field string) error {
	return errors.Join(
		ValidatePath(joinPath(field, "path"), r.Path, false),
		ValidateMethod(joinPath(field, "method"), r.Method, true),
	)
}
