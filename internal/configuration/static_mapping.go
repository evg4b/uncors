package configuration

import (
	"reflect"

	"github.com/mitchellh/mapstructure"
)

type StaticDirMappings = []StaticDirMapping

type StaticDirMapping struct {
	Path    string `mapstructure:"path"`
	Dir     string `mapstructure:"dir"`
	Default string `mapstructure:"default"`
}

func (s StaticDirMapping) Clone() StaticDirMapping {
	return StaticDirMapping{
		Path:    s.Path,
		Dir:     s.Dir,
		Default: s.Default,
	}
}

var staticDirMappingsType = reflect.TypeOf(StaticDirMappings{})

func StaticDirMappingHookFunc() mapstructure.DecodeHookFunc { //nolint: ireturn
	return func(f reflect.Type, t reflect.Type, rawData any) (any, error) {
		if t != staticDirMappingsType || f.Kind() != reflect.Map {
			return rawData, nil
		}

		if mappingsDefs, ok := rawData.(map[string]any); ok {
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

		return rawData, nil
	}
}
