//go:build integration

package integration

import (
	"regexp"
	"strings"
)

var (
	fullyScrubbed = regexp.MustCompile(`(?im)^(User-Agent|Date|Last-Modified):.*$`)

	// ephemeralPort matches an ephemeral ":port" (4–5 digits) so it can be
	// stabilised while the host it belongs to stays visible. Standard ports
	// (:80, :443) are 2–3 digits and are deliberately left intact.
	ephemeralPort = regexp.MustCompile(`:\d{4,5}\b`)
)

func Normalize(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")

	raw = fullyScrubbed.ReplaceAllStringFunc(raw, func(line string) string {
		name := line[:strings.IndexByte(line, ':')]

		return name + ": <scrubbed>"
	})

	return ephemeralPort.ReplaceAllString(raw, ":<port>")
}
