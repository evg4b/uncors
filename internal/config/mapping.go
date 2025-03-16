package config

import (
	"reflect"

	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"
)

type Mapping struct {
	From            string            `mapstructure:"from"`
	To              string            `mapstructure:"to"`
	Statics         StaticDirectories `mapstructure:"statics"`
	Mocks           Mocks             `mapstructure:"mocks"`
	Cache           CacheGlobs        `mapstructure:"cache"`
	Rewrites        RewriteOptions    `mapstructure:"rewrites"`
	OptionsHandling OptionsHandling   `mapstructure:"options-handling"`
}

func (m *Mapping) Clone() Mapping {
	return Mapping{
		From:            m.From,
		To:              m.To,
		Statics:         m.Statics.Clone(),
		Mocks:           m.Mocks.Clone(),
		Cache:           m.Cache.Clone(),
		Rewrites:        m.Rewrites.Clone(),
		OptionsHandling: m.OptionsHandling.Clone(),
	}
}

var (
	mappingType   = reflect.TypeOf(Mapping{})
	mappingFields = getTagValues(mappingType, "mapstructure")
)

func URLMappingHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, rawData any) (any, error) {
		if t != mappingType || f.Kind() != reflect.Map {
			return rawData, nil
		}

		if data, ok := rawData.(map[string]any); ok {
			availableFields, _ := lo.Difference(lo.Keys(data), mappingFields)
			if len(data) == 1 && len(availableFields) == 1 {
				return Mapping{
					From: availableFields[0],
					To:   data[availableFields[0]].(string), // nolint: forcetypeassert
				}, nil
			}

			mapping := Mapping{}
			err := decodeConfig(
				data,
				&mapping,
				StaticDirMappingHookFunc(),
			)

			return mapping, err
		}

		return rawData, nil
	}
}

func getTagValues(typeValue reflect.Type, tag string) []string {
	fields := reflect.VisibleFields(typeValue)

	return lo.FilterMap(fields, func(field reflect.StructField, _ int) (string, bool) {
		return field.Tag.Lookup(tag)
	})
}
