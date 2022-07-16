package main

import (
	"crypto/tls"
	"flag"
	"net"
	"net/http"
	"strconv"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/evg4b/uncors/internal/processor"
	"github.com/evg4b/uncors/internal/proxy"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

func main() {
	target := flag.String("target", "https://github.com", "Real target url (include https://)")
	source := flag.String("source", "http://localhost", "Local source url (include http://)")
	port := flag.Int("port", 3000, "Local listening port (3000 by default)")

	flag.Parse()

	factory, err := urlreplacer.NewUrlReplacerFactory(map[string]string{(*source): (*target)})
	if err != nil {
		pterm.Fatal.Println(err)
		return
	}

	proxyMiddleware := proxy.NewProxyHandlingMiddleware(
		proxy.WithUrlReplacerFactory(factory),
		proxy.WithHttpClient(http.Client{
			CheckRedirect: func(r *http.Request, v []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}),
	)

	rp := processor.NewRequestProcessor(
		processor.WithMiddleware(proxyMiddleware),
	)

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
	pterm.Println()

	http.HandleFunc("/", infrastructure.NormalizeHttpReqDecorator(rp.HandleRequest))
	addr := net.JoinHostPort("0.0.0.0", strconv.Itoa(*port))
	if err = http.ListenAndServe(addr, nil); err != nil {
		pterm.Fatal.Println(err)
		return
	}
}
