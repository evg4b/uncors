// nolint: cyclop
package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/evg4b/uncors/internal/server"
	"golang.org/x/net/context"

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

const baseAddress = "127.0.0.1"

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

	ctx := context.Background()

	uncorsServer := server.NewUncorsServer(ctx, mockMiddleware)
	finisher.Add(uncorsServer)
	go func() {
		log.Debugf("Starting http server on port %d", config.HTTPPort)
		addr := net.JoinHostPort(baseAddress, strconv.Itoa(config.HTTPPort))
		err := uncorsServer.ListenAndServe(addr)
		handleHTTPServerError("HTTP", err)
		finisher.Trigger()
	}()

	if config.IsHTTPSEnabled() {
		log.Debug("Found cert file and key file. Https server will be started")
		addr := net.JoinHostPort(baseAddress, strconv.Itoa(config.HTTPSPort))
		go func() {
			log.Debugf("Starting https server on port %d", config.HTTPSPort)
			err := uncorsServer.ListenAndServeTLS(addr, config.CertFile, config.KeyFile)
			handleHTTPServerError("HTTPS", err)
			finisher.Trigger()
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

func handleHTTPServerError(serverName string, err error) {
	if err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error(err)
		} else {
			log.Debugf("%s server was stopped with error %s", serverName, err)
		}
	} else {
		log.Debugf("%s server was stopped without errors", serverName)
	}
}
