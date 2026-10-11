package helpers

import (
	"net/url"
	"strings"
)

// SanitizeLogValue neutralises user-controlled input before it is written to a
// log record, preventing log forging through embedded line breaks. Carriage
// returns and line feeds are replaced with spaces.
func SanitizeLogValue(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")

	return strings.ReplaceAll(value, "\n", " ")
}

// LogURL renders a request URL for a log record: without user info, so a
// password in it never reaches the log, and sanitised like SanitizeLogValue.
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

// LogError renders an error for a log record, sanitised like SanitizeLogValue.
func LogError(err error) string {
	if err == nil {
		return ""
	}

	return SanitizeLogValue(err.Error())
}
