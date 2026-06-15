//go:build integration

package integration

import (
	"regexp"
	"strings"
)

var (
	// fullyScrubbed matches header lines whose entire value is non-deterministic
	// across runs (wall-clock time, the Go HTTP client version, file mtimes).
	fullyScrubbed = regexp.MustCompile(`(?im)^(User-Agent|Date|Last-Modified):.*$`)

	// ephemeralPort matches an ephemeral ":port" (4–5 digits) so it can be
	// stabilised while the host it belongs to stays visible. Standard ports
	// (:80, :443) are 2–3 digits and are deliberately left intact.
	ephemeralPort = regexp.MustCompile(`:\d{4,5}\b`)
)

// Normalize makes a raw HTTP request or response dump snapshot-stable while
// keeping it informative:
//
//   - Date and User-Agent values become <scrubbed>.
//   - Ephemeral ports become :<port>, so a rewritten "Host: 127.0.0.1:50653"
//     stays readable as "Host: 127.0.0.1:<port>" — the host rewriting the proxy
//     performed remains visible in the snapshot.
//   - Header order, casing, the start-line and the body are kept verbatim.
//   - Line endings are normalised to "\n".
func Normalize(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")

	raw = fullyScrubbed.ReplaceAllStringFunc(raw, func(line string) string {
		name := line[:strings.IndexByte(line, ':')]

		return name + ": <scrubbed>"
	})

	return ephemeralPort.ReplaceAllString(raw, ":<port>")
}
