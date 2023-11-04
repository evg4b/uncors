package validators

import (
	"fmt"
	"strings"
)

func joinPath(paths ...string) string {
	builder := strings.Builder{}
	prevPath := ""
	for _, path := range paths {
		if prevPath != "" && !strings.HasPrefix(path, "[") {
			builder.WriteString(".")
		}

		builder.WriteString(path)
		prevPath = path
	}

	return builder.String()
}

func index(i int) string {
	return fmt.Sprintf("[%d]", i)
}
