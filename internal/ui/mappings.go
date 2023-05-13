package ui

import (
	"fmt"
	"strings"

	"github.com/evg4b/uncors/internal/configuration"
)

func Mappings(mappings []configuration.URLMapping, mocksDefs []configuration.Mock) string {
	var builder strings.Builder

	for _, mapping := range mappings {
		if strings.HasPrefix(mapping.From, "https:") {
			builder.WriteString(fmt.Sprintf("PROXY: %s => %s\n", mapping.From, mapping.To))
			for _, static := range mapping.Statics {
				builder.WriteString(fmt.Sprintf("      static: %s => %s\n", static.Path, static.Dir))
			}
		}
	}
	for _, mapping := range mappings {
		if strings.HasPrefix(mapping.From, "http:") {
			builder.WriteString(fmt.Sprintf("PROXY: %s => %s\n", mapping.From, mapping.To))
			for _, static := range mapping.Statics {
				builder.WriteString(fmt.Sprintf("      static: %s => %s\n", static.Path, static.Dir))
			}
		}
	}
	if len(mocksDefs) > 0 {
		builder.WriteString(fmt.Sprintf("MOCKS: %d mock(s) registered", len(mocksDefs)))
	}

	builder.WriteString("\n")

	return builder.String()
}
