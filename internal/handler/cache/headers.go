package cache

import (
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/go-http-utils/headers"
)

const (
	anyValue         = "*"
	corsHeaderPrefix = "Access-Control-"
)

func isCorsHeader(name string) bool {
	return len(name) >= len(corsHeaderPrefix) && strings.EqualFold(name[:len(corsHeaderPrefix)], corsHeaderPrefix)
}

func forbidsStoring(header http.Header) bool {
	for _, value := range header[headers.CacheControl] {
		for directive := range strings.SplitSeq(value, ",") {
			name, _, _ := strings.Cut(directive, "=")
			name = strings.TrimSpace(name)

			if strings.EqualFold(name, "no-store") || strings.EqualFold(name, "private") {
				return true
			}
		}
	}

	return false
}

func variesOnAll(header http.Header) bool {
	for _, value := range header[headers.Vary] {
		for token := range strings.SplitSeq(value, ",") {
			if strings.TrimSpace(token) == anyValue {
				return true
			}
		}
	}

	return false
}

func varyNames(header http.Header) []string {
	var names []string

	for _, value := range header[headers.Vary] {
		for token := range strings.SplitSeq(value, ",") {
			name := http.CanonicalHeaderKey(strings.TrimSpace(token))

			switch name {
			case "", anyValue, headers.Origin, headers.AcceptEncoding:
				continue
			}

			if !slices.Contains(names, name) {
				names = append(names, name)
			}
		}
	}

	slices.Sort(names)

	return names
}

func variantKey(baseKey string, names []string, header http.Header) string {
	if len(names) == 0 {
		return baseKey
	}

	var builder strings.Builder

	builder.WriteString(baseKey)

	for _, name := range names {
		builder.WriteString("#")
		builder.WriteString(name)
		builder.WriteString("=")
		builder.WriteString(strconv.Quote(strings.Join(header.Values(name), ", ")))
	}

	return builder.String()
}

func acceptEncoding(header http.Header) string {
	var tokens []string

	for _, value := range header.Values(headers.AcceptEncoding) {
		for token := range strings.SplitSeq(value, ",") {
			token = strings.ToLower(strings.Join(strings.Fields(token), ""))
			if token != "" && !slices.Contains(tokens, token) {
				tokens = append(tokens, token)
			}
		}
	}

	slices.Sort(tokens)

	return strings.Join(tokens, ",")
}
