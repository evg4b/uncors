package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
)

var (
	ErrNoToPair   = errors.New("`to` values are not set for every `from`")
	ErrNoFromPair = errors.New("`from` values are not set for every `to`")
)

// mergeURLMappings merges from/to CLI pairs into cfg.Mappings.
// If a from URL already exists in the mappings, its to value is updated.
// Otherwise a new mapping entry is appended.
func mergeURLMappings(cfg *UncorsConfig, from, to []string) error {
	if len(from) > len(to) {
		return ErrNoToPair
	}

	if len(to) > len(from) {
		return ErrNoFromPair
	}

	for index, key := range from {
		found := false

		for i := range cfg.Mappings {
			if strings.EqualFold(cfg.Mappings[i].From, key) {
				cfg.Mappings[i].To = to[index]
				found = true

				break
			}
		}

		if !found {
			cfg.Mappings = append(cfg.Mappings, Mapping{
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

// NormaliseMappings normalises the From URL in each mapping: adds the default
// scheme (http) if absent and removes the port when it equals the scheme default.
func NormaliseMappings(mappings Mappings) Mappings {
	processedMappings := make(Mappings, 0, len(mappings))

	for _, mapping := range mappings {
		host, portStr, err := mapping.GetFromHostPort()
		if err != nil {
			panic(fmt.Errorf("failed to get host and port: %w", err))
		}

		sourceURL, err := mapping.GetFromURL()
		if err != nil {
			panic(fmt.Errorf("failed to parse source url: %w", err))
		}

		normalizedMapping := mapping.Clone()
		normalizedMapping.From = normalizeURL(*sourceURL, host, portStr)
		processedMappings = append(processedMappings, normalizedMapping)
	}

	return processedMappings
}

func normalizeURL(parsedURL url.URL, host, portStr string) string {
	scheme := parsedURL.Scheme
	if scheme == "" {
		scheme = httpScheme
	}

	var port int

	if portStr != "" {
		var err error

		port, err = strconv.Atoi(portStr)
		if err != nil {
			panic(fmt.Errorf("invalid port number: %w", err))
		}
	} else {
		if scheme == httpsScheme {
			port = defaultHTTPSPort
		} else {
			port = defaultHTTPPort
		}
	}

	parsedURL.Scheme = scheme

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
