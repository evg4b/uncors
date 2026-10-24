package config

import (
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/evg4b/uncors/pkg/urlt"
)

type Mappings []Mapping

type PortGroup struct {
	Port     int
	Scheme   string
	Mappings Mappings
}

type PortGroups []PortGroup

func (m Mappings) String() string {
	lines := make([]string, 0, len(m))

	for _, mapping := range m {
		lines = append(lines, fmt.Sprintf("%s => %s", mapping.From, mapping.To))

		for _, mock := range mapping.Mocks {
			lines = append(lines, fmt.Sprintf("    mock: %s", mock.String()))
		}

		for _, static := range mapping.Statics {
			lines = append(lines, fmt.Sprintf("    static: %s", static.String()))
		}

		for _, cacheGlob := range mapping.Cache {
			lines = append(lines, fmt.Sprintf("    cache: %s", cacheGlob))
		}
	}

	return strings.Join(lines, "\n")
}

func (m Mappings) GroupByPort() (PortGroups, error) {
	type portKey struct {
		port   int
		scheme string
	}

	grouped := make(map[portKey]Mappings)

	for _, mapping := range m {
		port, err := listenPort(mapping.From)
		if err != nil {
			return nil, fmt.Errorf("invalid port number: %w", err)
		}

		key := portKey{port: port, scheme: mapping.From.Scheme}
		grouped[key] = append(grouped[key], mapping)
	}

	result := make(PortGroups, 0, len(grouped))
	for key, mappings := range grouped {
		result = append(result, PortGroup{
			Port:     key.port,
			Scheme:   key.scheme,
			Mappings: mappings,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Port != result[j].Port {
			return result[i].Port < result[j].Port
		}

		return result[i].Scheme < result[j].Scheme
	})

	return result, nil
}

func listenPort(host urlt.Host) (int, error) {
	if host.Port != "" {
		return strconv.Atoi(host.Port)
	}

	if host.Scheme == httpsScheme {
		return defaultHTTPSPort, nil
	}

	return defaultHTTPPort, nil
}

func validateListeners(mappings Mappings) error {
	type listener struct {
		scheme string
		field  string
	}

	listeners := make(map[int]listener, len(mappings))
	sources := make(map[string]string, len(mappings))

	var errs []error

	for i, mapping := range mappings {
		field := joinPath("mappings", index(i), "from")
		from := normalizeHost(mapping.From)

		port, err := listenPort(from)
		if err != nil || from.Hostname == "" {
			continue
		}

		if owner, ok := listeners[port]; !ok {
			listeners[port] = listener{scheme: from.Scheme, field: field}
		} else if owner.scheme != from.Scheme {
			errs = append(errs, &ValidationError{fmt.Sprintf(
				"%s: port %d is already used for %s by %s", field, port, owner.scheme, owner.field,
			)})

			continue
		}

		source := strings.ToLower(from.Scheme + "://" + net.JoinHostPort(from.Hostname, strconv.Itoa(port)))
		if owner, ok := sources[source]; ok {
			errs = append(errs, &ValidationError{fmt.Sprintf(
				"%s: %s is already mapped by %s", field, mapping.From.String(), owner,
			)})

			continue
		}

		sources[source] = field
	}

	return errors.Join(errs...)
}
