package ui

import (
	"strings"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

func Logo(version string) string {
	logoLength := 51
	versionLine := strings.Repeat(" ", logoLength)
	versionSuffix := helpers.Sprintf("version: %s", version)
	versionPrefix := versionLine[:logoLength-len(versionSuffix)]

	logo, _ := pterm.DefaultBigText.
		WithLetters(
			putils.LettersFromStringWithStyle("UN", pterm.NewStyle(pterm.FgRed)),
			putils.LettersFromStringWithRGB("CORS", pterm.NewRGB(255, 215, 0)), //nolint: gomnd
		).
		Srender()

	var builder strings.Builder

	helpers.Fprintln(&builder)
	helpers.Fprint(&builder, logo)
	helpers.Fprint(&builder, versionPrefix, versionSuffix)
	helpers.Fprintln(&builder)

	return builder.String()
}
