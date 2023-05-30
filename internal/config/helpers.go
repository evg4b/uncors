package config

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/evg4b/uncors/internal/config/hooks"
	"github.com/evg4b/uncors/internal/sfmt"
	"github.com/evg4b/uncors/pkg/urlx"
	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"
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
		prev, ok := lo.Find(configuration.Mappings, func(item Mapping) bool {
			return strings.EqualFold(item.From, key)
		})

		if ok {
			// log.Warningf("Mapping for %s from (%s) replaced new value (%s)", key, prev, value)
			prev.To = value
		} else {
			configuration.Mappings = append(configuration.Mappings, Mapping{
				From: key,
				To:   value,
			})
		}
	}

	return nil
}

func decodeConfig[T any](data any, mapping *T, decodeFuncs ...mapstructure.DecodeHookFunc) error {
	hook := mapstructure.ComposeDecodeHookFunc(
		hooks.StringToTimeDurationHookFunc(),
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

const (
	httpScheme  = "http"
	httpsScheme = "https"
)

func NormaliseMappings(mappings Mappings, httpPort, httpsPort int, useHTTPS bool) (Mappings, error) {
	var processedMappings Mappings
	for _, mapping := range mappings {
		sourceURL, err := urlx.Parse(mapping.From)
		if err != nil {
			return nil, sfmt.Errorf("failed to parse source url: %w", err)
		}

		if isApplicableScheme(sourceURL.Scheme, httpScheme) {
			httpMapping := mapping.Clone()
			httpMapping.From = assignPortAndScheme(*sourceURL, httpScheme, httpPort)
			processedMappings = append(processedMappings, httpMapping)
		}

		if useHTTPS && isApplicableScheme(sourceURL.Scheme, httpsScheme) {
			httpsMapping := mapping.Clone()
			httpsMapping.From = assignPortAndScheme(*sourceURL, httpsScheme, httpsPort)
			processedMappings = append(processedMappings, httpsMapping)
		}
	}

	return processedMappings, nil
}

func assignPortAndScheme(parsedURL url.URL, scheme string, port int) string {
	host, _, _ := urlx.SplitHostPort(&parsedURL)
	parsedURL.Scheme = scheme

	if !(isDefaultPort(scheme, port)) {
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

func isApplicableScheme(scheme, expectedScheme string) bool {
	return strings.EqualFold(scheme, expectedScheme) || len(scheme) == 0
}
