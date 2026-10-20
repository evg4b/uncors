package config

import (
	"errors"
	"fmt"
	"path"
	"reflect"

	"github.com/samber/lo"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type StaticDirectory struct {
	Path  string `yaml:"path"`
	Dir   string `yaml:"dir"`
	Index string `yaml:"index"`
}

func (s *StaticDirectory) Clone() StaticDirectory {
	return StaticDirectory{
		Path:  s.Path,
		Dir:   s.Dir,
		Index: s.Index,
	}
}

func (s *StaticDirectory) String() string {
	return fmt.Sprintf("%s => %s", s.Path, s.Dir)
}

type StaticDirectories []StaticDirectory

func (s *StaticDirectories) Clone() StaticDirectories {
	if s == nil || *s == nil {
		return nil
	}

	return lo.Map(*s, func(item StaticDirectory, _ int) StaticDirectory {
		return item.Clone()
	})
}

type staticDirectoriesAlias StaticDirectories

type staticMapEntry struct {
	Dir   string `yaml:"dir"`
	Index string `yaml:"index"`
}

func (s *StaticDirectories) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(value.Content); i += 2 {
			dirPath := value.Content[i].Value
			valNode := value.Content[i+1]

			var staticDir StaticDirectory

			if valNode.Kind == yaml.ScalarNode {
				staticDir = StaticDirectory{Path: dirPath, Dir: valNode.Value}
			} else {
				err := valNode.Decode(&staticDir)
				if err != nil {
					return err
				}

				staticDir.Path = dirPath
			}

			*s = append(*s, staticDir)
		}

		return nil
	}

	return value.Decode((*staticDirectoriesAlias)(s))
}

func (s *StaticDirectories) checkKnownFields(field string, node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return checkSequenceItems(field, node, reflect.TypeFor[StaticDirectory]())
	}

	var errs []error

	for i := 0; i+1 < len(node.Content); i += 2 {
		errs = append(errs, checkKnownFields(
			joinPath(field, node.Content[i].Value),
			node.Content[i+1],
			reflect.TypeFor[staticMapEntry](),
		))
	}

	return errors.Join(errs...)
}

func (s *StaticDirectory) Validate(field string, fs afero.Fs) error {
	var errs []error

	errs = append(errs, ValidatePath(joinPath(field, "path"), s.Path, false))
	errs = append(errs, ValidateDirectory(joinPath(field, "dir"), s.Dir, fs))

	if s.Index != "" {
		errs = append(errs, ValidateFile(joinPath(field, "index"), path.Join(s.Dir, s.Index), fs))
	}

	return errors.Join(errs...)
}
