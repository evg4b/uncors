//go:build integration

package integration

import (
	"regexp"
	"strings"
)

// volatileHeader matches header lines whose values change between runs
// (timestamps, ports, software versions). Name and casing are preserved
// so ordering/casing regressions still show up in snapshots.
var volatileHeader = regexp.MustCompile(
	`(?im)^(Host|X-Forwarded-For|X-Forwarded-Host|X-Forwarded-Port|User-Agent|Date):.*$`,
)

// Normalize makes a raw HTTP request or response dump snapshot-stable:
// ephemeral header values are replaced with <scrubbed>, while header order,
// casing, the start-line and body are kept verbatim.
// Line endings are normalised to "\n".
func Normalize(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")

	return volatileHeader.ReplaceAllStringFunc(raw, func(line string) string {
		name := line[:strings.IndexByte(line, ':')]

		return name + ": <scrubbed>"
	})
}
