package config

import (
	"fmt"

	"github.com/samber/lo"
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

// UnmarshalYAML allows StaticDirectories to be specified as a YAML mapping
// (shorthand: path → dir or path → {dir, index}) as well as a sequence of
// full StaticDirectory objects.
//
// Map form:
//
//	statics:
//	  /path: /static-dir
//	  /other: { dir: /other-dir, index: index.html }
//
// Sequence form:
//
//	statics:
//	  - path: /path
//	    dir:  /static-dir
func (s *StaticDirectories) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(value.Content); i += 2 {
			path := value.Content[i].Value
			valNode := value.Content[i+1]

			var staticDir StaticDirectory

			if valNode.Kind == yaml.ScalarNode {
				staticDir = StaticDirectory{Path: path, Dir: valNode.Value}
			} else {
				err := valNode.Decode(&staticDir)
				if err != nil {
					return err
				}

				staticDir.Path = path // map key always wins over any inline path field
			}

			*s = append(*s, staticDir)
		}

		return nil
	}

	type staticDirectoriesAlias StaticDirectories

	return value.Decode((*staticDirectoriesAlias)(s))
}
