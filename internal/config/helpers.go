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

		fromHost, err := urlt.ParseHost(key)
		if err != nil {
			return fmt.Errorf("invalid `from` value %q: %w", key, err)
		}

		found := false

		for i := range cfg.Mappings {
			if sameSource(cfg.Mappings[i].From, *fromHost) {
				cfg.Mappings[i].To = *toHost
				found = true

				break
			}
		}

		if !found {
			cfg.Mappings = append(cfg.Mappings, Mapping{
				From: *fromHost,
				To:   *toHost,
			})
		}
	}

	return nil
}

func sameSource(left, right urlt.Host) bool {
	left = normalizeHost(left)
	right = normalizeHost(right)

	return strings.EqualFold(left.Scheme, right.Scheme) &&
		strings.EqualFold(left.Hostname, right.Hostname) &&
		left.Port == right.Port
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
