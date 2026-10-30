package helpers

import (
	"net/url"
	"strings"
)

func SanitizeLogValue(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")

	return strings.ReplaceAll(value, "\n", " ")
}

func LogURL(requestURL *url.URL) string {
	if requestURL == nil {
		return ""
	}

	redacted := url.URL{
		Scheme:   requestURL.Scheme,
		Host:     requestURL.Host,
		Path:     requestURL.Path,
		RawPath:  requestURL.RawPath,
		RawQuery: requestURL.RawQuery,
		Fragment: requestURL.Fragment,
	}

	return SanitizeLogValue(redacted.String())
}
