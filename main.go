// nolint: cyclop
package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/evg4b/uncors/internal/handler"

	"github.com/evg4b/uncors/internal/helpers"

	"github.com/evg4b/uncors/internal/version"

	"github.com/evg4b/uncors/internal/server"
	"golang.org/x/net/context"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/log"
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
	defer infra.PanicInterceptor(func(value any) {
		pterm.Error.Println(value)
		os.Exit(1)
	})

	pflag.Usage = func() {
		ui.Logo(Version)
		fmt.Fprintf(os.Stdout, "Usage of %s:\n", os.Args[0])
		pflag.PrintDefaults()
	}

	uncorsConfig, err := config.LoadConfiguration(viper.GetViper(), os.Args)
	if err != nil {
		panic(err)
	}

	if err = config.Validate(uncorsConfig); err != nil {
		panic(err)
	}

	if uncorsConfig.Debug {
		log.EnableDebugMessages()
		log.Debug("Enabled debug messages")
	}

	mappings, err := helpers.NormaliseMappings(
		uncorsConfig.Mappings,
		uncorsConfig.HTTPPort,
		uncorsConfig.HTTPSPort,
		uncorsConfig.IsHTTPSEnabled(),
	)
	if err != nil {
		panic(err)
	}

	factory, err := urlreplacer.NewURLReplacerFactory(mappings)
	if err != nil {
		panic(err)
	}

	httpClient, err := infra.MakeHTTPClient(viper.GetString("proxy"))
	if err != nil {
		panic(err)
	}

	finisher := finish.Finisher{Log: infra.NoopLogger{}}

	ctx := context.Background()

	globalHandler := handler.NewUncorsRequestHandler(
		handler.WithMappings(mappings),
		handler.WithLogger(ui.MockLogger),
		handler.WithFileSystem(afero.NewOsFs()),
		handler.WithURLReplacerFactory(factory),
		handler.WithHTTPClient(httpClient),
	)
	uncorsServer := server.NewUncorsServer(ctx, globalHandler)

	finisher.Add(uncorsServer)
	go func() {
		log.Debugf("Starting http server on port %d", uncorsConfig.HTTPPort)
		addr := net.JoinHostPort(baseAddress, strconv.Itoa(uncorsConfig.HTTPPort))
		err := uncorsServer.ListenAndServe(addr)
		handleHTTPServerError("HTTP", err)
		finisher.Trigger()
	}()

	if uncorsConfig.IsHTTPSEnabled() {
		log.Debug("Found cert file and key file. Https server will be started")
		addr := net.JoinHostPort(baseAddress, strconv.Itoa(uncorsConfig.HTTPSPort))
		go func() {
			log.Debugf("Starting https server on port %d", uncorsConfig.HTTPSPort)
			err := uncorsServer.ListenAndServeTLS(addr, uncorsConfig.CertFile, uncorsConfig.KeyFile)
			handleHTTPServerError("HTTPS", err)
			finisher.Trigger()
		}()
	}

	log.Print(ui.Logo(Version))
	log.Print("\n")
	log.Warning(ui.DisclaimerMessage)
	log.Print("\n")
	log.Info(mappings.String())
	log.Print("\n")

	go version.CheckNewVersion(ctx, httpClient, Version)

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
