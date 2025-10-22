package config

import (
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/samber/lo"
)

type Script struct {
	Matcher RequestMatcher `mapstructure:",squash"`
	Script  string         `mapstructure:"script"`
	File    string         `mapstructure:"file"`
}

func (s *Script) Clone() Script {
	return Script{
		Matcher: s.Matcher.Clone(),
		Script:  s.Script,
		File:    s.File,
	}
}

func (s *Script) String() string {
	method := "*"
	if s.Matcher.Method != "" {
		method = s.Matcher.Method
	}

	scriptType := "inline"
	if s.File != "" {
		scriptType = "file: " + s.File
	}

	return helpers.Sprintf("[%s script:%s] %s", method, scriptType, s.Matcher.Path)
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
