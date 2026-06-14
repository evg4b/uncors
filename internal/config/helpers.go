package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/evg4b/uncors/pkg/urlt"
)

var (
	ErrNoToPair   = errors.New("`to` values are not set for every `from`")
	ErrNoFromPair = errors.New("`from` values are not set for every `to`")
)

func mergeURLMappings(cfg *UncorsConfig, from, to []string) error {
	if len(from) > len(to) {
		return ErrNoToPair
	}

	if len(to) > len(from) {
		return ErrNoFromPair
	}

	for index, key := range from {
		toHost, err := urlt.ParseHost(to[index])
		if err != nil {
			return fmt.Errorf("invalid `to` value %q: %w", to[index], err)
		}

		found := false

		for i := range cfg.Mappings {
			if strings.EqualFold(cfg.Mappings[i].From.String(), key) {
				cfg.Mappings[i].To = *toHost
				found = true

				break
			}
		}

		if !found {
			fromHost, err := urlt.ParseHost(key)
			if err != nil {
				return fmt.Errorf("invalid `from` value %q: %w", key, err)
			}

			cfg.Mappings = append(cfg.Mappings, Mapping{
				From: *fromHost,
				To:   *toHost,
			})
		}
	}

	return nil
}

const (
	httpScheme  = "http"
	httpsScheme = "https"
)

func NormaliseMappings(mappings Mappings) Mappings {
	processedMappings := make(Mappings, 0, len(mappings))

	for _, mapping := range mappings {
		normalizedMapping := mapping.Clone()
		normalizedMapping.From = normalizeHost(mapping.From)
		processedMappings = append(processedMappings, normalizedMapping)
	}

	return processedMappings
}

// normalizeHost canonicalises a host: it forces the default http scheme when
// none is set and drops the port when it matches the scheme's default port.
func normalizeHost(host urlt.Host) urlt.Host {
	if host.Scheme == "" {
		host.Scheme = httpScheme
	}

	if host.Port != "" {
		port, err := strconv.Atoi(host.Port)
		if err == nil && isDefaultPort(host.Scheme, port) {
			host.Port = ""
		}
	}

	return host
}

func isDefaultPort(scheme string, port int) bool {
	return strings.EqualFold(httpScheme, scheme) && port == defaultHTTPPort ||
		strings.EqualFold(httpsScheme, scheme) && port == defaultHTTPSPort
}
