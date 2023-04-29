package configuration

import (
	"errors"
	"reflect"

	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"
)

type URLMapping struct {
	From string `mapstructure:"from"`
	To   string `mapstructure:"to"`
}

const FromKey = "from"
const ToKey = "to"

var urlMappingType = reflect.TypeOf(URLMapping{})
var urlMappingFields = getTagValues(urlMappingType, "mapstructure")

var ErrNoRequiredFields = errors.New("fields 'from' and 'to' are required")
var ErrFromShouldBeStrign = errors.New("fields 'from' and 'to' are required")
var ErrToShouldBeStrign = errors.New("fields 'from' and 'to' are required")

func URLMappingHookFunc() mapstructure.DecodeHookFunc { //nolint: ireturn
	return func(f reflect.Type, t reflect.Type, rawData any) (any, error) {
		if t != urlMappingType || f.Kind() != reflect.Map {
			return rawData, nil
		}

		data, ok := rawData.(map[string]any)
		if !ok {
			return rawData, nil
		}

		actualKeys := lo.Keys(data)
		incorrectFiles, missedFields := lo.Difference(actualKeys, urlMappingFields)
		if len(data) == 1 && len(incorrectFiles) == 1 {
			from := incorrectFiles[0]
			to, ok := data[incorrectFiles[0]].(string)
			if !ok {
				return nil, ErrToShouldBeStrign
			}

			return URLMapping{From: from, To: to}, nil
		}

		if lo.Contains(missedFields, FromKey) || lo.Contains(missedFields, ToKey) {
			return nil, ErrNoRequiredFields
		}

		from, ok := data[FromKey].(string)
		if !ok {
			return nil, ErrFromShouldBeStrign
		}

		to, ok := data[ToKey].(string)
		if !ok {
			return nil, ErrToShouldBeStrign
		}

		return URLMapping{From: from, To: to}, nil
	}
}

func getTagValues(typeValue reflect.Type, tag string) []string {
	var fields []string
	lo.ForEach(reflect.VisibleFields(typeValue), func(field reflect.StructField, index int) {
		if tag, ok := field.Tag.Lookup(tag); ok {
			fields = append(fields, tag)
		}
	})

	return fields
}
