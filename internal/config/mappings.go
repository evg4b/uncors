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
	var lines []string

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
	host, _, err := item.GetFromHostPort()
	if err != nil {
		panic(err)
	}

	return host
}

func (m Mappings) GroupByPort() PortGroups {
	type portKey struct {
		port   int
		scheme string
	}

	grouped := make(map[portKey]Mappings)

	for _, mapping := range m {
		_, portStr, err := mapping.GetFromHostPort()
		if err != nil {
			panic(fmt.Errorf("failed to get host and port: %w", err))
		}

		uri, err := mapping.GetFromURL()
		if err != nil {
			panic(fmt.Errorf("failed to parse mapping from URL: %w", err))
		}

		port := 80 // default HTTP port
		if portStr != "" {
			port, err = strconv.Atoi(portStr)
			if err != nil {
				panic(fmt.Errorf("invalid port number: %w", err))
			}
		} else if uri.Scheme == "https" {
			port = 443 // default HTTPS port
		}

		key := portKey{port: port, scheme: uri.Scheme}
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

	// Sort by port for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		if result[i].Port != result[j].Port {
			return result[i].Port < result[j].Port
		}

		return result[i].Scheme < result[j].Scheme
	})

	return result
}
