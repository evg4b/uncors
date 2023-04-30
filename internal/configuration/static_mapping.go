package configuration

import (
	"errors"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

type StaticDirMappings = []StaticDirMapping

type StaticDirMapping struct {
	Path    string `mapstructure:"path"`
	Dir     string `mapstructure:"folder"`
	Default string `mapstructure:"default"`
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

				if def, ok := mappingDef.(map[string]string); ok {
					mappings = append(mappings, StaticDirMapping{
						Path:    path,
						Dir:     def["dir"],
						Default: def["default"],
					})

					continue
				}

				return nil, errors.New("FAIL")
			}

			return mappings, nil
		}

		return rawData, nil
	}
}
