package config

import (
	"errors"

	"github.com/spf13/viper"
)

var (
	ErrNoToPair   = errors.New("`to` values are not set for every `from`")
	ErrNoFromPair = errors.New("`from` values are not set for every `to`")
)

func ReadURLMapping(config *viper.Viper) (map[string]string, error) {
	urlMappings := map[string]string{}
	from, to := config.GetStringSlice("from"), config.GetStringSlice("to") //nolint: varnamelen

	if len(from) > len(to) {
		return nil, ErrNoToPair
	}

	if len(to) > len(from) {
		return nil, ErrNoFromPair
	}

	for index, key := range from {
		urlMappings[key] = to[index]
	}

	return urlMappings, nil
}
