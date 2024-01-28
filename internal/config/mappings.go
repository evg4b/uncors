package config

import (
	"strings"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/pkg/urlx"
	"github.com/samber/lo"
)

type Mappings []Mapping

func (m Mappings) String() string {
	builder := &strings.Builder{}

	for _, group := range lo.GroupBy(m, extractHost) {
		for _, mapping := range group {
			helpers.FPrintf(builder, "%s => %s\n", mapping.From, mapping.To)
		}

		mapping := group[0]
		for _, mock := range mapping.Mocks {
			helpers.FPrintf(builder, "    mock: %s\n", mock.String())
		}
		for _, static := range mapping.Statics {
			helpers.FPrintf(builder, "    static: %s\n", static.String())
		}
		for _, cacheGlob := range mapping.Cache {
			helpers.FPrintf(builder, "    cache: %s\n", cacheGlob)
		}
	}

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
