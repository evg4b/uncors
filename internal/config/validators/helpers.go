package validators

import "strings"

func joinPath(paths ...string) string {
	builder := strings.Builder{}
	prevPath := ""
	for _, path := range paths {
		hasPrefix := strings.HasPrefix(path, "[")
		hasSuffix := strings.HasSuffix(prevPath, "]")
		if prevPath != "" && !hasSuffix && !hasPrefix {
			builder.WriteString(".")
		}

		builder.WriteString(path)
		prevPath = path
	}

	return builder.String()
}
