package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/samber/lo"
	"github.com/spf13/afero"
)

type Script struct {
	Matcher RequestMatcher `yaml:",inline"`
	Script  string         `yaml:"script"`
	File    string         `yaml:"file"`
}

// scriptMarshal is the canonical YAML representation of Script.
// Using a flat struct (no inline) and trimming multi-line script strings avoids
// a gopkg.in/yaml.v3 round-trip bug where strings starting with \n are
// serialized as "|4" block scalars with wrong content indentation.
type scriptMarshal struct {
	Path    string            `yaml:"path,omitempty"`
	Method  string            `yaml:"method,omitempty"`
	Queries map[string]string `yaml:"queries,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
	Script  string            `yaml:"script,omitempty"`
	File    string            `yaml:"file,omitempty"`
}

func (s Script) MarshalYAML() (any, error) {
	return scriptMarshal{
		Path:    s.Matcher.Path,
		Method:  s.Matcher.Method,
		Queries: s.Matcher.Queries,
		Headers: s.Matcher.Headers,
		Script:  strings.TrimSpace(s.Script),
		File:    s.File,
	}, nil
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

func (s *Script) Validate(field string, fs afero.Fs) error {
	var errs []error

	errs = append(errs, s.Matcher.Validate(field))

	switch {
	case s.Script == "" && s.File == "":
		scriptField := joinPath(field, "script")
		fileField := joinPath(field, "file")

		const neitherMsg = ": either 'script' or 'file' must be provided"

		errs = append(errs, &ValidationError{scriptField + neitherMsg})
		errs = append(errs, &ValidationError{fileField + neitherMsg})
	case s.Script != "" && s.File != "":
		scriptField := joinPath(field, "script")
		fileField := joinPath(field, "file")

		const bothMsg = ": only one of 'script' or 'file' can be provided"

		errs = append(errs, &ValidationError{scriptField + bothMsg})
		errs = append(errs, &ValidationError{fileField + bothMsg})
	case s.File != "":
		errs = append(errs, ValidateFile(joinPath(field, "file"), s.File, fs))
	}

	return errors.Join(errs...)
}
