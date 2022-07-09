package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/evg4b/uncors/inernal/handler"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

var (
	target   = "github.com"
	protocol = "https"
	origin   = "github.local.com:3000"
)

func main() {
	flag.StringVar(&target, "target", target, "host:port to proxy requests to")
	flag.StringVar(&protocol, "protocol", protocol, "protocol used by the target")
	flag.StringVar(&origin, "origin", origin, "origin header to be used for the proxy request")

	flag.Parse()

	logoLetters := []pterm.Letters{
		putils.LettersFromStringWithStyle("Un", pterm.NewStyle(pterm.FgRed)),
		putils.LettersFromStringWithRGB("CORS", pterm.NewRGB(255, 215, 0)),
	}

	uncorsLogo, _ := pterm.DefaultBigText.
		WithLetters(logoLetters...).
		Srender()

	pterm.Println()
	pterm.Print(uncorsLogo)
	pterm.Println()

	reqHandler := handler.NewRequestHandler(
		handler.WithOrigin(origin),
		handler.WithProtocol(protocol),
		handler.WithTarget(target),
	)

	http.HandleFunc("/", reqHandler.HandleRequest)

	log.Println(origin, "=>", target)
	http.ListenAndServe(":3000", nil)
}
