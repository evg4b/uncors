package hooks

import (
	"reflect"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
)

func StringToTimeDurationHookFunc() mapstructure.DecodeHookFunc { //nolint: ireturn
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}

		if t != reflect.TypeOf(time.Second) {
			return data, nil
		}

		trimmed := strings.ReplaceAll(data.(string), " ", "") //nolint: forcetypeassert

		return time.ParseDuration(trimmed) //nolint:wrapcheck
	}
}
