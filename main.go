// nolint: cyclop
package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/evg4b/uncors/internal/options"
	"github.com/evg4b/uncors/internal/processor"
	"github.com/evg4b/uncors/internal/proxy"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/pseidemann/finish"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

var Version = "X.X.X"

const (
	defaultHTTPPort  = 80
	defaultHTTPSPort = 443
	baseAddress      = "0.0.0.0"
)

func main() {
	target := flag.String("target", "https://github.com", "Target host with protocol for to the resource to be proxy")
	source := flag.String("source", "localhost", "Local host with protocol for to the resource from which proxying will take place") // nolint: lll
	httpPort := flag.Int("http-port", defaultHTTPPort, "Local HTTP listening port")
	httpsPort := flag.Int("https-port", defaultHTTPSPort, "Local HTTPS listening port")
	certFile := flag.String("cert-file", "", "Path to HTTPS certificate file")
	keyFile := flag.String("key-file", "", "Path to matching for certificate private key")
	proxyURL := flag.String("proxy", "", "HTTP/HTTPS proxy to provide requests to real server (used system by default)")

	flag.Usage = func() {
		printLogo()
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	mappings, err := urlreplacer.NormaliseMappings(
		map[string]string{*source: *target},
		*httpPort,
		*httpsPort,
		len(*certFile) > 0 && len(*keyFile) > 0,
	)
	if err != nil {
		pterm.Fatal.Println(err)
	}

	factory, err := urlreplacer.NewURLReplacerFactory(mappings)
	if err != nil {
		pterm.Fatal.Println(err)
	}

	httpClient, err := infrastructure.MakeHTTPClient(*proxyURL)
	if err != nil {
		pterm.Fatal.Println(err)
	}

	optionsMiddleware := options.NewOptionsMiddleware()
	proxyMiddleware := proxy.NewProxyMiddleware(
		proxy.WithURLReplacerFactory(factory),
		proxy.WithHTTPClient(httpClient),
	)

	requestProcessor := processor.NewRequestProcessor(
		processor.WithMiddleware(optionsMiddleware),
		processor.WithMiddleware(proxyMiddleware),
	)

	finisher := finish.Finisher{Log: infrastructure.NoopLogger{}}

	httpServer := infrastructure.NewServer(baseAddress, *httpPort, requestProcessor)
	finisher.Add(httpServer, finish.WithName("http"))
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			pterm.Error.Println(err)
		}
	}()

	if certFile != nil && keyFile != nil {
		httpsServer := infrastructure.NewServer(baseAddress, *httpsPort, requestProcessor)
		finisher.Add(httpsServer, finish.WithName("https"))
		go func() {
			if err := httpsServer.ListenAndServeTLS(*certFile, *keyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
				pterm.Error.Println(err)
			}
		}()
	}

	printLogo()
	printMappings(mappings)

	finisher.Wait()

	pterm.Info.Print("Server was stopped")
}

func printLogo() {
	logoLength := 51
	versionLine := strings.Repeat(" ", logoLength)
	versionSuffix := fmt.Sprintf("version: %s", Version)
	versionPrefix := versionLine[:logoLength-len(versionSuffix)]

	logo, _ := pterm.DefaultBigText.
		WithLetters(
			putils.LettersFromStringWithStyle("UN", pterm.NewStyle(pterm.FgRed)),
			putils.LettersFromStringWithRGB("CORS", pterm.NewRGB(255, 215, 0)), // nolint: gomnd
		).
		Srender()

	pterm.Println()
	pterm.Print(logo)
	pterm.Println(versionPrefix + versionSuffix)
	pterm.Println()
}

func printMappings(mappings map[string]string) {
	builder := strings.Builder{}
	for source, target := range mappings {
		if strings.HasPrefix(source, "https:") {
			builder.WriteString(fmt.Sprintf("PROXY: %s => %s\n", source, target))
		}
	}
	for source, target := range mappings {
		if strings.HasPrefix(source, "http:") {
			builder.WriteString(fmt.Sprintf("PROXY: %s => %s\n", source, target))
		}
	}
	pterm.Info.Printfln(builder.String())
}
