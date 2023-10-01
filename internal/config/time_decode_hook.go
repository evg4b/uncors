package config

import (
	"reflect"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
)

func StringToTimeDurationHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if f.Kind() != reflect.String || t != reflect.TypeOf(time.Second) {
			return data, nil
		}

		return time.ParseDuration(
			strings.ReplaceAll(data.(string), " ", ""), //nolint: forcetypeassert
		)
	}
}
