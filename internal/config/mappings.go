package config

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/samber/lo"
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

	for _, group := range lo.GroupBy(m, extractHost) {
		for _, mapping := range group {
			lines = append(lines, fmt.Sprintf("%s => %s", mapping.From, mapping.To))
		}

		mapping := lo.FirstOrEmpty(group)
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

func extractHost(item Mapping) string {
	return item.From.Hostname
}

func (m Mappings) GroupByPort() PortGroups {
	type portKey struct {
		port   int
		scheme string
	}

	grouped := make(map[portKey]Mappings)

	for _, mapping := range m {
		port := defaultHTTPPort
		portStr := mapping.From.Port

		if portStr != "" {
			parsedPort, err := strconv.Atoi(portStr)
			if err != nil {
				panic(fmt.Errorf("invalid port number: %w", err))
			}

			port = parsedPort
		} else if mapping.From.Scheme == httpsScheme {
			port = defaultHTTPSPort
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

	return result
}
