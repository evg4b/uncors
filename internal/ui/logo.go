package ui

import (
	"fmt"
	"strings"

	"github.com/evg4b/uncors/internal/log"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

func Logo(version string) string {
	logoLength := 51
	versionLine := strings.Repeat(" ", logoLength)
	versionSuffix := fmt.Sprintf("version: %s", version)
	versionPrefix := versionLine[:logoLength-len(versionSuffix)]

	logo, _ := pterm.DefaultBigText.
		WithLetters(
			putils.LettersFromStringWithStyle("UN", pterm.NewStyle(pterm.FgRed)),
			putils.LettersFromStringWithRGB("CORS", pterm.NewRGB(255, 215, 0)), //nolint: gomnd
		).
		Srender()

	var builder strings.Builder

	if _, err := fmt.Fprintln(&builder); err != nil {
		log.Fatal(err)
	}

	if _, err := fmt.Fprint(&builder, logo); err != nil {
		log.Fatal(err)
	}

	if _, err := fmt.Fprint(&builder, versionPrefix, versionSuffix); err != nil {
		log.Fatal(err)
	}

	if _, err := fmt.Fprintln(&builder); err != nil {
		log.Fatal(err)
	}

	return builder.String()
}
