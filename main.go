package main

import (
	"errors"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/log"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/sfmt"
	"github.com/evg4b/uncors/internal/ui"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/internal/version"
	goCache "github.com/patrickmn/go-cache"
	"github.com/pseidemann/finish"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
)

var Version = "X.X.X"

const baseAddress = "127.0.0.1"

func main() {
	defer infra.PanicInterceptor(func(value any) {
		log.Error(value)
		os.Exit(1)
	})

	pflag.Usage = func() {
		ui.Logo(Version)
		sfmt.Fprintf(os.Stdout, "Usage of %s:\n", os.Args[0])
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

	mappings, err := config.NormaliseMappings(
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

	cacheConfig := uncorsConfig.CacheConfig
	cacheStorage := goCache.New(cacheConfig.ExpirationTime, cacheConfig.ClearTime)

	globalHandler := handler.NewUncorsRequestHandler(
		handler.WithMappings(mappings),
		handler.WithLogger(ui.MockLogger),
		handler.WithFileSystem(afero.NewOsFs()),
		handler.WithURLReplacerFactory(factory),
		handler.WithHTTPClient(httpClient),
		handler.WithCacheMiddlewareFactory(func(key string, globs config.CacheGlobs) contracts.MiddlewareHandler {
			return cache.NewMiddleware(
				cache.WithLogger(ui.CacheLogger),
				cache.WithPrefix(key),
				cache.WithCacheStorage(cacheStorage),
				cache.WithGlobs(globs),
			)
		}),
	)

	finisher := finish.Finisher{Log: infra.NoopLogger{}}

	ctx := context.Background()

	uncorsServer := server.NewUncorsServer(ctx, globalHandler)

	log.Print(ui.Logo(Version))
	log.Print("\n")
	log.Warning(ui.DisclaimerMessage)
	log.Print("\n")
	log.Info(mappings.String())
	log.Print("\n")

	finisher.Add(uncorsServer)
	go func() {
		defer finisher.Trigger()
		log.Debugf("Starting http server on port %d", uncorsConfig.HTTPPort)
		addr := net.JoinHostPort(baseAddress, strconv.Itoa(uncorsConfig.HTTPPort))
		err := uncorsServer.ListenAndServe(addr)
		handleHTTPServerError("HTTP", err)
	}()

	if uncorsConfig.IsHTTPSEnabled() {
		log.Debug("Found cert file and key file. Https server will be started")
		addr := net.JoinHostPort(baseAddress, strconv.Itoa(uncorsConfig.HTTPSPort))
		go func() {
			defer finisher.Trigger()
			log.Debugf("Starting https server on port %d", uncorsConfig.HTTPSPort)
			err := uncorsServer.ListenAndServeTLS(addr, uncorsConfig.CertFile, uncorsConfig.KeyFile)
			handleHTTPServerError("HTTPS", err)
		}()
	}

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
