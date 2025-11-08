package config

import (
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
)

func StringToTimeDurationHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if f.Kind() != reflect.String || t != reflect.TypeFor[time.Duration]() {
			return data, nil
		}

		if value, ok := data.(string); ok {
			return time.ParseDuration(strings.ReplaceAll(value, " ", ""))
		}

		return nil, errors.ErrUnsupported
	}
}
