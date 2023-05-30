package config

import (
	"strings"

	"github.com/evg4b/uncors/internal/sfmt"
	"github.com/evg4b/uncors/pkg/urlx"
	"github.com/samber/lo"
)

type Mappings []Mapping

func (mappings Mappings) String() string {
	var builder strings.Builder

	groups := lo.GroupBy(mappings, extractHost)
	for _, group := range groups {
		for _, mapping := range group {
			builder.WriteString(sfmt.Sprintf("%s => %s\n", mapping.From, mapping.To))
		}

		mapping := group[0]
		for _, mock := range mapping.Mocks {
			builder.WriteString(sfmt.Sprintf("    mock: [%s %d] %s\n", mock.Method, mock.Response.Code, mock.Path))
		}
		for _, static := range mapping.Statics {
			builder.WriteString(sfmt.Sprintf("    static: %s => %s\n", static.Path, static.Dir))
		}
	}

	builder.WriteString("\n")

	return builder.String()
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
