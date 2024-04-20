package config

import (
	"reflect"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"
)

type StaticDirectory struct {
	Path  string `mapstructure:"path"`
	Dir   string `mapstructure:"dir"`
	Index string `mapstructure:"index"`
}

func (s *StaticDirectory) Clone() StaticDirectory {
	return StaticDirectory{
		Path:  s.Path,
		Dir:   s.Dir,
		Index: s.Index,
	}
}

func (s *StaticDirectory) String() string {
	return helpers.Sprintf("%s => %s", s.Path, s.Dir)
}

type StaticDirectories []StaticDirectory

func (s StaticDirectories) Clone() StaticDirectories {
	if s == nil {
		return nil
	}

	return lo.Map(s, func(item StaticDirectory, _ int) StaticDirectory {
		return item.Clone()
	})
}

var staticDirMappingsType = reflect.TypeOf(StaticDirectories{})

func StaticDirMappingHookFunc() mapstructure.DecodeHookFunc { //nolint: ireturn
	return func(f reflect.Type, t reflect.Type, rawData any) (any, error) {
		if t != staticDirMappingsType || f.Kind() != reflect.Map {
			return rawData, nil
		}

		mappingsDefs, ok := rawData.(map[string]any)
		if !ok {
			return rawData, nil
		}

		var mappings StaticDirectories
		for path, mappingDef := range mappingsDefs {
			if def, ok := mappingDef.(string); ok {
				mappings = append(mappings, StaticDirectory{
					Path: path,
					Dir:  def,
				})

				continue
			}

			mapping := StaticDirectory{}
			err := decodeConfig(mappingDef, &mapping)
			if err != nil {
				return nil, err
			}

			mapping.Path = path
			mappings = append(mappings, mapping)
		}

		return mappings, nil
	}
}
