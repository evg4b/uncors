package config

import "github.com/evg4b/uncors/internal/helpers"

type RequestMatcher struct {
	Path    string            `mapstructure:"path"`
	Method  string            `mapstructure:"method"`
	Queries map[string]string `mapstructure:"queries"`
	Headers map[string]string `mapstructure:"headers"`
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
