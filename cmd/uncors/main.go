package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/evg4b/uncors/internal/options"
	"github.com/evg4b/uncors/internal/processor"
	"github.com/evg4b/uncors/internal/proxy"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

var Version = "X.X.X"

const (
	defaulTimeout = 5 * time.Minute
	defaultPort   = 3000
)

func main() {
	target := flag.String("target", "https://github.com", "Real target url (include https://)")
	source := flag.String("source", "http://localhost", "Local source url (include http://)")
	port := flag.Int("port", defaultPort, "Local listening port (3000 by default)")

	flag.Parse()

	factory, err := urlreplacer.NewURLReplacerFactory(map[string]string{(*source): (*target)})
	if err != nil {
		pterm.Fatal.Println(err)

		return
	}

	optionsMiddleware := options.NewOptionsMiddleware()
	proxyMiddleware := proxy.NewProxyMiddleware(
		proxy.WithURLReplacerFactory(factory),
		proxy.WithHTTPClient(&http.Client{
			CheckRedirect: func(r *http.Request, v []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// nolint: gosec
					InsecureSkipVerify: true,
				},
			},
			Jar:     nil,
			Timeout: defaulTimeout,
		}),
	)

	requestProcessor := processor.NewRequestProcessor(
		processor.WithMiddleware(optionsMiddleware),
		processor.WithMiddleware(proxyMiddleware),
	)

	printLogo()
	pterm.Info.Printfln("PROXY: %s => %s", *source, *target)
	pterm.Println()
	http.HandleFunc("/", infrastructure.NormalizeHTTPReqDecorator(requestProcessor.HandleRequest))
	address := net.JoinHostPort("0.0.0.0", strconv.Itoa(*port))

	if err = http.ListenAndServe(address, nil); err != nil {
		pterm.Fatal.Println(err)
	}
}

func printLogo() {
	logoLength := 51
	versionLine := strings.Repeat(" ", logoLength)
	versionSuffix := fmt.Sprintf("version: %s", Version)
	versionPreffix := versionLine[:logoLength-len(versionSuffix)]

	logo, _ := pterm.DefaultBigText.
		WithLetters(
			putils.LettersFromStringWithStyle("UN", pterm.NewStyle(pterm.FgRed)),
			putils.LettersFromStringWithRGB("CORS", pterm.NewRGB(255, 215, 0)), // nolint: gomnd
		).
		Srender()

	pterm.Println()
	pterm.Print(logo)
	pterm.Println(versionPreffix + versionSuffix)
	pterm.Println()
}
