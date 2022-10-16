package ui

import (
	"fmt"
	"strings"

	"github.com/evg4b/uncors/internal/mock"
)

func Mappings(mappings map[string]string, mocksDefs []mock.Mock) string {
	var builder strings.Builder

	for source, target := range mappings {
		if strings.HasPrefix(source, "https:") {
			builder.WriteString(fmt.Sprintf("PROXY: %s => %s\n", source, target))
		}
	}
	for source, target := range mappings {
		if strings.HasPrefix(source, "http:") {
			builder.WriteString(fmt.Sprintf("PROXY: %s => %s\n", source, target))
		}
	}
	if len(mocksDefs) > 0 {
		builder.WriteString(fmt.Sprintf("MOCKS: %d mock(s) registered", len(mocksDefs)))
	}

	builder.WriteString("\n")

	return builder.String()
}
