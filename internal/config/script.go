package config

import (
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/samber/lo"
)

type Script struct {
	Path    string            `mapstructure:"path"`
	Method  string            `mapstructure:"method"`
	Queries map[string]string `mapstructure:"queries"`
	Headers map[string]string `mapstructure:"headers"`
	Script  string            `mapstructure:"script"`
	File    string            `mapstructure:"file"`
}

func (s *Script) Clone() Script {
	return Script{
		Path:    s.Path,
		Method:  s.Method,
		Queries: helpers.CloneMap(s.Queries),
		Headers: helpers.CloneMap(s.Headers),
		Script:  s.Script,
		File:    s.File,
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
