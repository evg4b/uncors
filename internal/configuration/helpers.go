package configuration

import (
	"errors"

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
		if prev, ok := configuration.Mappings[key]; ok {
			log.Warningf("Mapping for %s from (%s) replaced new value (%s)", key, prev, value)
		}

		configuration.Mappings[key] = value
	}

	return nil
}
