package main

import (
	"log"
	"net/http"

	"github.com/evg4b/uncors/inernal/processor"
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

	rp := processor.NewRequestProcessor()

	// reqHandler := handler.NewRequestHandler(
	// 	handler.WithOrigin(source),
	// 	handler.WithTarget(target),
	// 	handler.WithUrlReplcaer(urlreplacer.NewSimpleReplacer(map[string]string{
	// 		source: target,
	// 	})),
	// )

	http.HandleFunc("/", rp.HandleRequest)

	// http.HandleFunc("/", infrastrucure.NormalizeHttpReqDecorator(reqHandler.HandleRequest))
	log.Println(source, "=>", target)
	http.ListenAndServe(":3000", nil)
}
