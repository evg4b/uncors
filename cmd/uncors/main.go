package main

import (
	"log"
	"net/http"

	"github.com/evg4b/uncors/inernal/handler"
	"github.com/evg4b/uncors/inernal/urlreplacer"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

var (
	target = "github.com"
	source = "localhost:3000"
)

func main() {
	logoLetters := []pterm.Letters{
		putils.LettersFromStringWithStyle("UN", pterm.NewStyle(pterm.FgRed)),
		putils.LettersFromStringWithRGB("CORS", pterm.NewRGB(255, 215, 0)),
	}

	uncorsLogo, _ := pterm.DefaultBigText.
		WithLetters(logoLetters...).
		Srender()

	pterm.Println()
	pterm.Print(uncorsLogo)
	pterm.Println()

	reqHandler := handler.NewRequestHandler(
		handler.WithOrigin(source),
		handler.WithTarget(target),
		handler.WithUrlReplcaer(urlreplacer.NewSimpleReplacer(map[string]string{
			source: target,
		})),
	)

	http.HandleFunc("/", reqHandler.HandleRequest)
	log.Println(source, "=>", target)
	http.ListenAndServe(":3000", nil)
}
