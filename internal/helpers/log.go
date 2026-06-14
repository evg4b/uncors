package helpers

import "strings"

// SanitizeLogValue neutralises user-controlled input before it is written to a
// log entry, preventing log-forging via embedded line breaks. Carriage returns
// and line feeds are replaced with spaces so a single logical entry stays on a
// single line.
func SanitizeLogValue(value string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(value)
}
