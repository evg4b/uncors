package uncors

import (
	"strings"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

const (
	colorRed   = 255
	colorGreen = 215
	colorBlue  = 0
)

func Logo(version string) string {
	logoLength := 51
	versionLine := strings.Repeat(" ", logoLength)
	versionSuffix := helpers.Sprintf("version: %s", version)
	versionPrefix := versionLine[:logoLength-len(versionSuffix)]

	logo, _ := pterm.DefaultBigText.
		WithLetters(
			putils.LettersFromStringWithStyle("UN", pterm.NewStyle(pterm.FgRed)),
			putils.LettersFromStringWithRGB("CORS", pterm.NewRGB(colorRed, colorGreen, colorBlue)),
		).
		Srender()

	var builder strings.Builder

	helpers.FPrintln(&builder)
	helpers.FPrint(&builder, logo)
	helpers.FPrint(&builder, versionPrefix, versionSuffix)
	helpers.FPrintln(&builder)

	return builder.String()
}
