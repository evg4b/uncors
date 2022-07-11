package main

import (
	"flag"
	"net/http"

	"github.com/evg4b/uncors/internal/infrastrucure"
	"github.com/evg4b/uncors/internal/processor"
	"github.com/evg4b/uncors/internal/proxy"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

func main() {
	target := flag.String("target", "https://github.com", "Real target url (include https://)")
	source := flag.String("source", "http://localhost:3000", "Local source url (include http://)")

	flag.Parse()

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
	pterm.Info.Printfln("PROXY: %s => %s", *source, *target)

	proxyMiddleware := proxy.NewProxyHandlingMiddleware(
		proxy.WithUrlReplcaer(
			urlreplacer.NewSimpleReplacer(map[string]string{
				(*source): (*target),
			}),
		),
	)

	rp := processor.NewRequestProcessor(
		processor.WithMiddleware(proxyMiddleware),
	)

	http.HandleFunc("/", infrastrucure.NormalizeHttpReqDecorator(rp.HandleRequest))
	http.ListenAndServe(":3000", nil)
}
