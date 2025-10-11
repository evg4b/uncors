package config

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/evg4b/uncors/pkg/urlx"
	"github.com/samber/lo"
)

type Mappings []Mapping

// PortGroup represents mappings grouped by port
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

		mapping := group[0]
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
	uri, err := urlx.Parse(item.From)
	if err != nil {
		panic(err)
	}

	host, _, err := urlx.SplitHostPort(uri)
	if err != nil {
		panic(err)
	}

	return host
}

// GroupByPort groups mappings by their port and scheme
func (m Mappings) GroupByPort() PortGroups {
	type portKey struct {
		port   int
		scheme string
	}

	grouped := make(map[portKey]Mappings)

	for _, mapping := range m {
		uri, err := urlx.Parse(mapping.From)
		if err != nil {
			panic(fmt.Errorf("failed to parse mapping from URL: %w", err))
		}

		_, portStr, err := urlx.SplitHostPort(uri)
		if err != nil {
			panic(fmt.Errorf("failed to split host and port: %w", err))
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

	var result PortGroups
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
