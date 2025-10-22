package config

import (
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/samber/lo"
)

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

type Script struct {
	RequestMatcher `mapstructure:",squash"`
	Script         string `mapstructure:"script"`
	File           string `mapstructure:"file"`
}

func (s *Script) Clone() Script {
	return Script{
		RequestMatcher: s.RequestMatcher.Clone(),
		Script:         s.Script,
		File:           s.File,
	}
}

func (s *Script) String() string {
	method := "*"
	if s.Method != "" {
		method = s.Method
	}

	scriptType := "inline"
	if s.File != "" {
		scriptType = "file: " + s.File
	}

	return helpers.Sprintf("[%s script:%s] %s", method, scriptType, s.Path)
}

type Scripts []Script

func (s Scripts) Clone() Scripts {
	if s == nil {
		return nil
	}

	return lo.Map(s, func(item Script, _ int) Script {
		return item.Clone()
	})
}
