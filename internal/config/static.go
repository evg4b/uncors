package config

import (
	"reflect"

	"github.com/mitchellh/mapstructure"
)

type StaticDirs = []StaticDir

type StaticDir struct {
	Path  string `mapstructure:"path"`
	Dir   string `mapstructure:"dir"`
	Index string `mapstructure:"index"`
}

func (s StaticDir) Clone() StaticDir {
	return StaticDir{
		Path:  s.Path,
		Dir:   s.Dir,
		Index: s.Index,
	}
}

var staticDirMappingsType = reflect.TypeOf(StaticDirs{})

func StaticDirMappingHookFunc() mapstructure.DecodeHookFunc { //nolint: ireturn
	return func(f reflect.Type, t reflect.Type, rawData any) (any, error) {
		if t != staticDirMappingsType || f.Kind() != reflect.Map {
			return rawData, nil
		}

		mappingsDefs, ok := rawData.(map[string]any)
		if !ok {
			return rawData, nil
		}

		var mappings StaticDirs
		for path, mappingDef := range mappingsDefs {
			if def, ok := mappingDef.(string); ok {
				mappings = append(mappings, StaticDir{
					Path: path,
					Dir:  def,
				})

				continue
			}

			mapping := StaticDir{}
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
