package configuration

import (
	"errors"

	"github.com/spf13/viper"
)

var (
	ErrNoToPair   = errors.New("`to` values are not set for every `from`")
	ErrNoFromPair = errors.New("`from` values are not set for every `to`")
)

func readURLMapping(config *viper.Viper, configuration *UncorsConfig) error {
	from, to := config.GetStringSlice("from"), config.GetStringSlice("to") //nolint: varnamelen

	if len(from) > len(to) {
		return ErrNoToPair
	}

	if len(to) > len(from) {
		return ErrNoFromPair
	}

	for index, key := range from {
		configuration.Mappings[key] = to[index]
	}

	return nil
}
