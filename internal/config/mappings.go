package config

import (
	"fmt"
	"strings"

	"github.com/evg4b/uncors/pkg/urlx"
	"github.com/samber/lo"
)

type Mappings []Mapping

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
