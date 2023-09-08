package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	cf "github.com/evg4b/uncors/internal/config"
	c "github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/handler/mock"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/handler/static"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/log"
	"github.com/evg4b/uncors/internal/server"
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
	defer helpers.PanicInterceptor(func(value any) {
		log.Error(value)
		os.Exit(1)
	})

	pflag.Usage = func() {
		ui.Logo(Version)
		helpers.Fprintf(os.Stdout, "Usage of %s:\n", os.Args[0])
		pflag.PrintDefaults()
	}

	uncorsConfig := cf.LoadConfiguration(viper.GetViper(), os.Args)

	if err := cf.Validate(uncorsConfig); err != nil {
		panic(err)
	}

	if uncorsConfig.Debug {
		log.EnableDebugMessages()
		log.Debug("Enabled debug messages")
	}

	mappings := cf.NormaliseMappings(
		uncorsConfig.Mappings,
		uncorsConfig.HTTPPort,
		uncorsConfig.HTTPSPort,
		uncorsConfig.IsHTTPSEnabled(),
	)

	factory := urlreplacer.NewURLReplacerFactory(mappings)
	httpClient := infra.MakeHTTPClient(viper.GetString("proxy"))

	cacheConfig := uncorsConfig.CacheConfig
	cacheStorage := goCache.New(cacheConfig.ExpirationTime, cacheConfig.ClearTime)

	fs := afero.NewOsFs()
	globalHandler := handler.NewUncorsRequestHandler(
		handler.WithMappings(mappings),
		handler.WithLogger(ui.MockLogger),
		handler.WithCacheMiddlewareFactory(func(globs cf.CacheGlobs) c.Middleware {
			return cache.NewMiddleware(
				cache.WithLogger(ui.CacheLogger),
				cache.WithMethods(cacheConfig.Methods),
				cache.WithCacheStorage(cacheStorage),
				cache.WithGlobs(globs),
			)
		}),
		handler.WithProxyHandlerFactory(func() c.Handler {
			return proxy.NewProxyHandler(
				proxy.WithURLReplacerFactory(factory),
				proxy.WithHTTPClient(httpClient),
				proxy.WithLogger(ui.ProxyLogger),
			)
		}),
		handler.WithStaticHandlerFactory(func(path string, dir cf.StaticDirectory) c.Middleware {
			return static.NewStaticMiddleware(
				static.WithFileSystem(afero.NewBasePathFs(fs, dir.Dir)),
				static.WithIndex(dir.Index),
				static.WithLogger(ui.StaticLogger),
				static.WithPrefix(path),
			)
		}),
		handler.WithMockHandlerFactory(func(response cf.Response) c.Handler {
			return mock.NewMockHandler(
				mock.WithLogger(ui.MockLogger),
				mock.WithResponse(response),
				mock.WithFileSystem(fs),
				mock.WithAfter(time.After),
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
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		log.Debugf("%s server was stopped without errors", serverName)
	} else {
		panic(fmt.Errorf("%s server was stopped with error %w", serverName, err))
	}
}
