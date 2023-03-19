// nolint: cyclop
package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/evg4b/uncors/internal/configuration"
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
)

var Version = "X.X.X"

const baseAddress = "0.0.0.0"

func main() {
	defer infrastructure.PanicInterceptor(func(value any) {
		pterm.Error.Println(value)
		os.Exit(1)
	})

	pflag.Usage = func() {
		ui.Logo(Version)
		fmt.Fprintf(os.Stdout, "Usage of %s:\n", os.Args[0])
		pflag.PrintDefaults()
	}

	config, err := configuration.LoadConfiguration(viper.GetViper(), os.Args)
	if err != nil {
		panic(err)
	}

	if err = configuration.Validate(config); err != nil {
		panic(err)
	}

	if config.Debug {
		viper.Debug()
		log.EnableDebugMessages()
		log.Debug("Enabled debug messages")
	}

	mappings, err := urlreplacer.NormaliseMappings(
		config.Mappings,
		config.HTTPPort,
		config.HTTPSPort,
		config.IsHTTPSEnabled(),
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
		mock.WithMocks(config.Mocks),
		mock.WithFileSystem(fileSystem),
	)

	finisher := finish.Finisher{Log: infrastructure.NoopLogger{}}

	httpServer := infrastructure.NewServer(baseAddress, config.HTTPPort, mockMiddleware)
	finisher.Add(httpServer, finish.WithName("http"))
	go func() {
		log.Debugf("Starting http server on port %d", config.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error(err)
		}
	}()

	if config.IsHTTPSEnabled() {
		log.Debug("Found cert file and key file. Https server will be started")
		httpsServer := infrastructure.NewServer(baseAddress, config.HTTPSPort, mockMiddleware)
		finisher.Add(httpsServer, finish.WithName("https"))
		go func() {
			log.Debugf("Starting https server on port %d", config.HTTPSPort)
			err := httpsServer.ListenAndServeTLS(config.CertFile, config.KeyFile)
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Error(err)
			}
		}()
	}

	log.Print(ui.Logo(Version))
	log.Print("\n")
	log.Warning(ui.DisclaimerMessage)
	log.Print("\n")
	log.Info(ui.Mappings(mappings, config.Mocks))
	log.Print("\n")

	go ui.CheckLastVersion(httpClient, Version)

	finisher.Wait()

	log.Info("Server was stopped")
}
