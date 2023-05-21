package config

import (
	"reflect"

	"github.com/mitchellh/mapstructure"
)

type StaticDirMappings = []StaticDirMapping

type StaticDirMapping struct {
	Path  string `mapstructure:"path"`
	Dir   string `mapstructure:"dir"`
	Index string `mapstructure:"index"`
}

func (s StaticDirMapping) Clone() StaticDirMapping {
	return StaticDirMapping{
		Path:  s.Path,
		Dir:   s.Dir,
		Index: s.Index,
	}
}

var staticDirMappingsType = reflect.TypeOf(StaticDirMappings{})

func StaticDirMappingHookFunc() mapstructure.DecodeHookFunc { //nolint: ireturn
	return func(f reflect.Type, t reflect.Type, rawData any) (any, error) {
		if t != staticDirMappingsType || f.Kind() != reflect.Map {
			return rawData, nil
		}

		mappingsDefs, ok := rawData.(map[string]any)
		if !ok {
			return rawData, nil
		}

		var mappings StaticDirMappings
		for path, mappingDef := range mappingsDefs {
			if def, ok := mappingDef.(string); ok {
				mappings = append(mappings, StaticDirMapping{
					Path: path,
					Dir:  def,
				})

				continue
			}

			mapping := StaticDirMapping{}
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
