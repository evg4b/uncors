package configuration

import (
	"reflect"

	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"
)

type URLMapping struct {
	From    string            `mapstructure:"from"`
	To      string            `mapstructure:"to"`
	Statics StaticDirMappings `mapstructure:"statics"`
}

var urlMappingType = reflect.TypeOf(URLMapping{})
var urlMappingFields = getTagValues(urlMappingType, "mapstructure")

func URLMappingHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, rawData any) (any, error) {
		if t != urlMappingType || f.Kind() != reflect.Map {
			return rawData, nil
		}

		if data, ok := rawData.(map[string]any); ok {
			availableFields, _ := lo.Difference(lo.Keys(data), urlMappingFields)
			if len(data) == 1 && len(availableFields) == 1 {
				return URLMapping{
					From: availableFields[0],
					To:   data[availableFields[0]].(string), // nolint: forcetypeassert
				}, nil
			}

			return decodeConfig(data, URLMapping{})
		}

		return rawData, nil
	}
}

func getTagValues(typeValue reflect.Type, tag string) []string {
	fields := reflect.VisibleFields(typeValue)

	return lo.FilterMap(fields, func(field reflect.StructField, index int) (string, bool) {
		return field.Tag.Lookup(tag)
	})
}
