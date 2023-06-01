package config

import (
	"reflect"

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

type StaticDirectories []StaticDirectory

func (d StaticDirectories) Clone() StaticDirectories {
	if d == nil {
		return nil
	}

	return lo.Map(d, func(item StaticDirectory, index int) StaticDirectory {
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
