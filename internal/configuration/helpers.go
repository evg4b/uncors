package configuration

import (
	"errors"
	"strings"

	"github.com/mitchellh/mapstructure"

	"github.com/samber/lo"

	"github.com/evg4b/uncors/internal/log"

	"github.com/spf13/viper"
)

var (
	ErrNoToPair   = errors.New("`to` values are not set for every `from`")
	ErrNoFromPair = errors.New("`from` values are not set for every `to`")
)

func readURLMapping(config *viper.Viper, configuration *UncorsConfig) error {
	from, to := config.GetStringSlice("from"), config.GetStringSlice("to")

	if len(from) > len(to) {
		return ErrNoToPair
	}

	if len(to) > len(from) {
		return ErrNoFromPair
	}

	for index, key := range from {
		value := to[index]
		prev, ok := lo.Find(configuration.Mappings, func(item URLMapping) bool {
			return strings.EqualFold(item.From, key)
		})

		if ok {
			log.Warningf("Mapping for %s from (%s) replaced new value (%s)", key, prev, value)
			prev.To = value
		} else {
			configuration.Mappings = append(configuration.Mappings, URLMapping{
				From: key,
				To:   value,
			})
		}
	}

	return nil
}

func decodeConfig[T any](data any, mapping *T, decodeFuncs ...mapstructure.DecodeHookFunc) error {
	hook := mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToSliceHookFunc(","),
		mapstructure.ComposeDecodeHookFunc(decodeFuncs...),
	)
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:               mapping,
		DecodeHook:           hook,
		ErrorUnused:          true,
		IgnoreUntaggedFields: true,
	})

	if err != nil {
		return err //nolint:wrapcheck
	}

	err = decoder.Decode(data)

	return err //nolint:wrapcheck
}
