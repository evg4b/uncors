package configuration

import (
	"errors"
	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"
	"reflect"
)

type URLMapping struct {
	From  string   `mapstructure:"from"`
	To    string   `mapstructure:"to"`
	Mocks []string `mapstructure:"mocks"`
}

var urlMappingType = reflect.TypeOf(URLMapping{})
var urlMappingFields = getTagValues()

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
			return URLMapping{
				From:  incorrectFiles[0],
				To:    data[incorrectFiles[0]].(string),
				Mocks: []string{},
			}, nil
		}

		if lo.Contains(missedFields, "from") || lo.Contains(missedFields, "to") {
			return nil, errors.New("dlkasjdl")
		}

		mocks := []string{}
		if !lo.Contains(missedFields, "mocks") {
			mocks = lo.Map(data["mocks"].([]any), func(item any, index int) string {
				return item.(string)
			})
		}

		return URLMapping{
			From:  data["from"].(string),
			To:    data["to"].(string),
			Mocks: mocks,
		}, nil
	}
}

func getTagValues() []string {
	fields := []string{}
	typeValue := urlMappingType
	for i := 0; i < typeValue.NumField(); i++ {
		field := typeValue.Field(i)
		if tag, ok := field.Tag.Lookup("mapstructure"); ok {
			fields = append(fields, tag)
		}
	}
	return fields
}
