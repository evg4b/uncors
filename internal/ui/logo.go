package ui

import (
	"strings"

	"github.com/evg4b/uncors/internal/sfmt"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

func Logo(version string) string {
	logoLength := 51
	versionLine := strings.Repeat(" ", logoLength)
	versionSuffix := sfmt.Sprintf("version: %s", version)
	versionPrefix := versionLine[:logoLength-len(versionSuffix)]

	logo, _ := pterm.DefaultBigText.
		WithLetters(
			putils.LettersFromStringWithStyle("UN", pterm.NewStyle(pterm.FgRed)),
			putils.LettersFromStringWithRGB("CORS", pterm.NewRGB(255, 215, 0)), //nolint: gomnd
		).
		Srender()

	var builder strings.Builder

	sfmt.Fprintln(&builder)
	sfmt.Fprint(&builder, logo)
	sfmt.Fprint(&builder, versionPrefix, versionSuffix)
	sfmt.Fprintln(&builder)

	return builder.String()
}
