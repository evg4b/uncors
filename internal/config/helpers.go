package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/evg4b/uncors/pkg/urlx"
	"github.com/mitchellh/mapstructure"
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
		found := false
		for i := range configuration.Mappings {
			if strings.EqualFold(configuration.Mappings[i].From, key) {
				configuration.Mappings[i].To = to[index]
				found = true

				break
			}
		}

		if !found {
			configuration.Mappings = append(configuration.Mappings, Mapping{
				From: key,
				To:   to[index],
			})
		}
	}

	return nil
}

func decodeConfig[T any](data any, mapping *T, decodeFuncs ...mapstructure.DecodeHookFunc) error {
	hook := mapstructure.ComposeDecodeHookFunc(
		StringToTimeDurationHookFunc(),
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
		return err
	}

	err = decoder.Decode(data)

	return err
}

const (
	httpScheme  = "http"
	httpsScheme = "https"
)

func NormaliseMappings(mappings Mappings) Mappings {
	processedMappings := Mappings{}
	for _, mapping := range mappings {
		sourceURL, err := urlx.Parse(mapping.From)
		if err != nil {
			panic(fmt.Errorf("failed to parse source url: %w", err))
		}

		// Normalize the mapping with port from URL
		normalizedMapping := mapping.Clone()
		normalizedMapping.From = normalizeURL(*sourceURL)
		processedMappings = append(processedMappings, normalizedMapping)
	}

	return processedMappings
}

func normalizeURL(parsedURL url.URL) string {
	host, portStr, err := urlx.SplitHostPort(&parsedURL)
	if err != nil {
		panic(fmt.Errorf("failed to split host and port: %w", err))
	}

	// Determine the scheme (default to http if not specified)
	scheme := parsedURL.Scheme
	if scheme == "" {
		scheme = httpScheme
	}

	// Parse port or use default based on scheme
	var port int
	if portStr != "" {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			panic(fmt.Errorf("invalid port number: %w", err))
		}
	} else {
		// Use default port based on scheme
		if scheme == httpsScheme {
			port = defaultHTTPSPort
		} else {
			port = defaultHTTPPort
		}
	}

	parsedURL.Scheme = scheme

	// Only include port in host if it's not the default port for the scheme
	if !isDefaultPort(scheme, port) {
		parsedURL.Host = net.JoinHostPort(host, strconv.Itoa(port))
	} else {
		parsedURL.Host = host
	}

	return parsedURL.String()
}

func isDefaultPort(scheme string, port int) bool {
	return strings.EqualFold(httpScheme, scheme) && port == defaultHTTPPort ||
		strings.EqualFold(httpsScheme, scheme) && port == defaultHTTPSPort
}
