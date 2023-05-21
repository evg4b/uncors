package config

import (
	"fmt"
	"strings"
)

type Mappings []Mapping

func (mappings Mappings) String() string {
	var builder strings.Builder

	for _, mapping := range mappings {
		if strings.HasPrefix(mapping.From, "https:") {
			builder.WriteString(fmt.Sprintf("PROXY: %s => %s\n", mapping.From, mapping.To))
			for _, static := range mapping.Statics {
				builder.WriteString(fmt.Sprintf("      static: %s => %s\n", static.Path, static.Dir))
			}
		}
		if len(mapping.Mocks) > 0 {
			builder.WriteString(fmt.Sprintf("MOCKS: %d mock(s) registered", len(mapping.Mocks)))
		}
	}
	for _, mapping := range mappings {
		if strings.HasPrefix(mapping.From, "http:") {
			builder.WriteString(fmt.Sprintf("PROXY: %s => %s\n", mapping.From, mapping.To))
			for _, static := range mapping.Statics {
				builder.WriteString(fmt.Sprintf("      static: %s => %s\n", static.Path, static.Dir))
			}
		}
		if len(mapping.Mocks) > 0 {
			builder.WriteString(fmt.Sprintf("MOCKS: %d mock(s) registered", len(mapping.Mocks)))
		}
	}

	builder.WriteString("\n")

	return builder.String()
}
