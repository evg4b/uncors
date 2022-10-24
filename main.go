// nolint: cyclop
package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/evg4b/uncors/internal/log"
	"github.com/evg4b/uncors/internal/mock"
	"github.com/evg4b/uncors/internal/proxy"
	"github.com/evg4b/uncors/internal/ui"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/gorilla/mux"
	"github.com/pseidemann/finish"
	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
)

var Version = "X.X.X"

const (
	defaultHTTPPort  = 80
	defaultHTTPSPort = 443
	baseAddress      = "0.0.0.0"
)

var proxyLogger = log.NewLogger(" PROXY ", log.WithStyle(&pterm.Style{
	pterm.FgBlack,
	pterm.BgLightBlue,
}))

var mockLogger = log.NewLogger(" MOCK  ", log.WithStyle(&pterm.Style{
	pterm.FgBlack,
	pterm.BgLightMagenta,
}))

func main() {
	target := flag.String("target", "https://github.com", "Target host with protocol for to the resource to be proxy")
	source := flag.String("source", "localhost", "Local host with protocol for to the resource from which proxying will take place") // nolint: lll
	httpPort := flag.Int("http-port", defaultHTTPPort, "Local HTTP listening port")
	httpsPort := flag.Int("https-port", defaultHTTPSPort, "Local HTTPS listening port")
	certFile := flag.String("cert-file", "", "Path to HTTPS certificate file")
	keyFile := flag.String("key-file", "", "Path to matching for certificate private key")
	proxyURL := flag.String("proxy", "", "HTTP/HTTPS proxy to provide requests to real server (used system by default)")
	mocksFile := flag.String("mocks", "", "File with configured mocks")
	debug := flag.Bool("debug", false, "Show debug output")

	flag.Usage = func() {
		ui.Logo(Version)
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if *debug {
		log.EnableDebugMessages()
	}

	router := mux.NewRouter()

	var mocksDefs []mock.Mock
	if len(*mocksFile) > 0 {
		file, err := os.Open(*mocksFile)
		if err != nil {
			log.Fatal(err)
		}

		decoder := yaml.NewDecoder(file)
		if err = decoder.Decode(&mocksDefs); err != nil {
			log.Fatal(err)
		}
	}

	mock.MakeMockedRoutes(router, mockLogger, mocksDefs)

	mappings, err := urlreplacer.NormaliseMappings(
		map[string]string{*source: *target},
		*httpPort,
		*httpsPort,
		len(*certFile) > 0 && len(*keyFile) > 0,
	)
	if err != nil {
		log.Fatal(err)
	}

	factory, err := urlreplacer.NewURLReplacerFactory(mappings)
	if err != nil {
		log.Fatal(err)
	}

	httpClient, err := infrastructure.MakeHTTPClient(*proxyURL)
	if err != nil {
		log.Fatal(err)
	}

	proxyHandler := proxy.NewProxyHandler(
		proxy.WithURLReplacerFactory(factory),
		proxy.WithHTTPClient(httpClient),
		proxy.WithLogger(proxyLogger),
	)

	router.NotFoundHandler = proxyHandler
	router.MethodNotAllowedHandler = proxyHandler

	finisher := finish.Finisher{Log: infrastructure.NoopLogger{}}

	httpServer := infrastructure.NewServer(baseAddress, *httpPort, router)
	finisher.Add(httpServer, finish.WithName("http"))
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error(err)
		}
	}()

	if len(*certFile) > 0 && len(*keyFile) > 0 {
		httpsServer := infrastructure.NewServer(baseAddress, *httpsPort, router)
		finisher.Add(httpsServer, finish.WithName("https"))
		go func() {
			if err := httpsServer.ListenAndServeTLS(*certFile, *keyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Error(err)
			}
		}()
	}

	log.Print(ui.Logo(Version))
	log.Info(ui.Mappings(mappings, mocksDefs))

	finisher.Wait()

	log.Info("Server was stopped")
}
