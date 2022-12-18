// nolint: cyclop
package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/evg4b/uncors/internal/log"
	"github.com/evg4b/uncors/internal/middlewares/mock"
	"github.com/evg4b/uncors/internal/middlewares/proxy"
	"github.com/evg4b/uncors/internal/ui"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/pseidemann/finish"
	"github.com/pterm/pterm"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var Version = "X.X.X"

const (
	defaultHTTPPort  = 80
	defaultHTTPSPort = 443
	baseAddress      = "0.0.0.0"
)

func main() {
	pflag.StringSlice("to", []string{}, "Target host with protocol for to the resource to be proxy")
	pflag.StringSlice("from", []string{}, "Local host with protocol for to the resource from which proxying will take place") //nolint: lll
	pflag.Int("http-port", defaultHTTPPort, "Local HTTP listening port")
	pflag.Int("https-port", defaultHTTPSPort, "Local HTTPS listening port")
	pflag.String("cert-file", "", "Path to HTTPS certificate file")
	pflag.String("key-file", "", "Path to matching for certificate private key")
	pflag.String("proxy", "", "HTTP/HTTPS proxy to provide requests to real server (used system by default)")
	pflag.String("mocks", "", "File with configured mocks")
	pflag.Bool("debug", false, "Show debug output")

	defer infrastructure.PanicInterceptor(func(value interface{}) {
		pterm.Error.Println(value)
		os.Exit(1)
	})

	pflag.Usage = func() {
		ui.Logo(Version)
		fmt.Fprintf(os.Stdout, "Usage of %s:\n", os.Args[0])
		pflag.PrintDefaults()
	}

	pflag.Parse()
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		panic(err)
	}

	httpPort := viper.GetInt("http-port")
	httpsPort := viper.GetInt("https-port")
	certFile := viper.GetString("cert-file")
	keyFile := viper.GetString("key-file")
	mocksFile := viper.GetString("mocks")

	if viper.GetBool("debug") {
		viper.Debug()
		log.EnableDebugMessages()
		log.Debug("Enabled debug messages")
	}

	var mocksDefs []mock.Mock
	if len(mocksFile) > 0 {
		file, err := os.Open(mocksFile)
		if err != nil {
			panic(err)
		}

		log.Debugf("Loaded file with mocks '%s'", mocksFile)
		decoder := yaml.NewDecoder(file)
		if err = decoder.Decode(&mocksDefs); err != nil {
			panic(err)
		}
	}

	urlMappings, err := config.ReadURLMapping(viper.GetViper())
	if err != nil {
		panic(err)
	}
	mappings, err := urlreplacer.NormaliseMappings(
		urlMappings,
		httpPort,
		httpsPort,
		len(certFile) > 0 && len(keyFile) > 0,
	)
	if err != nil {
		panic(err)
	}

	factory, err := urlreplacer.NewURLReplacerFactory(mappings)
	if err != nil {
		panic(err)
	}

	httpClient, err := infrastructure.MakeHTTPClient(viper.GetString("proxy"))
	if err != nil {
		panic(err)
	}

	proxyMiddleware := proxy.NewProxyMiddleware(
		proxy.WithURLReplacerFactory(factory),
		proxy.WithHTTPClient(httpClient),
		proxy.WithLogger(ui.ProxyLogger),
	)

	fileSystem := afero.NewOsFs()

	mockMiddleware := mock.NewMockMiddleware(
		mock.WithLogger(ui.MockLogger),
		mock.WithNextMiddleware(proxyMiddleware),
		mock.WithMocks(mocksDefs),
		mock.WithFileSystem(fileSystem),
	)

	finisher := finish.Finisher{Log: infrastructure.NoopLogger{}}

	httpServer := infrastructure.NewServer(baseAddress, httpPort, mockMiddleware)
	finisher.Add(httpServer, finish.WithName("http"))
	go func() {
		log.Debugf("Starting http server on port %d", httpPort)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error(err)
		}
	}()

	if len(certFile) > 0 && len(keyFile) > 0 {
		log.Debug("Found cert file and key file. Https server will be started")
		httpsServer := infrastructure.NewServer(baseAddress, httpsPort, mockMiddleware)
		finisher.Add(httpsServer, finish.WithName("https"))
		go func() {
			log.Debugf("Starting https server on port %d", httpsPort)
			if err := httpsServer.ListenAndServeTLS(certFile, keyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Error(err)
			}
		}()
	}

	log.Print(ui.Logo(Version))
	log.Print("\n")
	log.Warning(ui.DisclaimerMessage)
	log.Print("\n")
	log.Info(ui.Mappings(mappings, mocksDefs))
	log.Print("\n")

	go ui.CheckLastVersion(httpClient, Version)

	finisher.Wait()

	log.Info("Server was stopped")
}
