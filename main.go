package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/evg4b/uncors/internal/options"
	"github.com/evg4b/uncors/internal/processor"
	"github.com/evg4b/uncors/internal/proxy"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/urlreplacer"
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

	flag.Usage = func() {
		printLogo()
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(
		ctx,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	defer cancel()

	flag.Parse()

	mappings := map[string]string{
		*source: *target,
	}

	factory, err := urlreplacer.NewURLReplacerFactory(mappings)
	if err != nil {
		pterm.Fatal.Println(err)

		return
	}

	optionsMiddleware := options.NewOptionsMiddleware()
	proxyMiddleware := proxy.NewProxyMiddleware(
		proxy.WithURLReplacerFactory(factory),
		proxy.WithHTTPClient(&infrastructure.HTTPClient),
	)

	requestProcessor := processor.NewRequestProcessor(
		processor.WithMiddleware(optionsMiddleware),
		processor.WithMiddleware(proxyMiddleware),
	)

	uncorsServer := server.NewServer(
		server.WithHTTP(baseAddress, *httpPort),
		server.WithHTTPS(baseAddress, *httpsPort),
		server.WithSslCert(*certFile),
		server.WithSslKey(*keyFile),
		server.WithRequestProcessor(requestProcessor),
	)

	printLogo()
	printMappings(mappings, *httpPort, *httpsPort, len(*certFile) > 0 && len(*keyFile) > 0)

	if err = uncorsServer.ListenAndServe(ctx); err != nil {
		pterm.Fatal.Println(err)
	} else {
		pterm.Info.Print("Server was stopped")
	}
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

func printMappings(mappings map[string]string, port int, httpsPort int, hasHTTPS bool) {
	builder := strings.Builder{}
	for source, target := range mappings {
		if port == defaultHTTPPort {
			builder.WriteString(fmt.Sprintf("PROXY: http://%s => %s\n", source, target))
		} else {
			builder.WriteString(fmt.Sprintf("PROXY: http://%s:%d => %s\n", source, port, target))
		}
		if hasHTTPS {
			if httpsPort == defaultHTTPSPort {
				builder.WriteString(fmt.Sprintf("PROXY: https://%s => %s\n", source, target))
			} else {
				builder.WriteString(fmt.Sprintf("PROXY: https://%s:%d => %s\n", source, httpsPort, target))
			}
		}
	}
	pterm.Info.Printfln(builder.String())
}
