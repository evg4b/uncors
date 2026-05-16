package config

import (
	"fmt"

	"github.com/samber/lo"
	"github.com/spf13/afero"
)

type Script struct {
	Matcher RequestMatcher `yaml:",inline"`
	Script  string         `yaml:"script"`
	File    string         `yaml:"file"`
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

	return fmt.Sprintf("[%s script:%s] %s", method, scriptType, s.Matcher.Path)
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

func (s *Script) Validate(field string, fs afero.Fs, errs *Errors) {
	s.Matcher.Validate(field, errs)

	switch {
	case s.Script == "" && s.File == "":
		scriptField := joinPath(field, "script")
		fileField := joinPath(field, "file")

		errs.add(fmt.Sprintf("%s: either 'script' or 'file' must be provided", scriptField))
		errs.add(fmt.Sprintf("%s: either 'script' or 'file' must be provided", fileField))
	case s.Script != "" && s.File != "":
		scriptField := joinPath(field, "script")
		fileField := joinPath(field, "file")

		errs.add(fmt.Sprintf("%s: only one of 'script' or 'file' can be provided", scriptField))
		errs.add(fmt.Sprintf("%s: only one of 'script' or 'file' can be provided", fileField))
	case s.File != "":
		ValidateFile(joinPath(field, "file"), s.File, fs, errs)
	}
}
