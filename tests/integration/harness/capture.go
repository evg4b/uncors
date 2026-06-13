//go:build integration

package harness

import (
	"regexp"
	"strings"
)

// volatileLine matches header lines whose values are ephemeral across runs
// (random ports, client version, timestamps). The header name and casing are
// preserved so ordering/casing regressions still surface in snapshots.
var volatileLine = regexp.MustCompile(
	`(?im)^(Host|X-Forwarded-For|X-Forwarded-Host|X-Forwarded-Port|User-Agent|Date):.*$`,
)

// Normalize makes a raw request dump snapshot-stable: ephemeral header values
// are replaced with <scrubbed>, while header order, casing, the request line and
// the body are kept verbatim. Line endings are normalised to "\n".
//
// Tune the scrubbing per suite: if a test specifically asserts a value such as
// X-Forwarded-Proto, drop that header from volatileLine so it stays in the snapshot.
func Normalize(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")

	return volatileLine.ReplaceAllStringFunc(raw, func(line string) string {
		name := line[:strings.IndexByte(line, ':')]

		return name + ": <scrubbed>"
	})
}
