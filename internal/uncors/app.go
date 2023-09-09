package uncors

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"syscall"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/handler/mock"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/handler/static"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/log"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/ui"
	"github.com/evg4b/uncors/internal/urlreplacer"
	goCache "github.com/patrickmn/go-cache"
	"github.com/pseidemann/finish"
	"github.com/spf13/afero"
	"golang.org/x/net/context"
)

type App struct {
	fs          afero.Fs
	version     string
	baseAddress string
	finisher    finish.Finisher
}

func CreateApp(fs afero.Fs, version string, baseAddress string) *App {
	return &App{
		fs:          fs,
		version:     version,
		baseAddress: baseAddress,
		finisher: finish.Finisher{
			Log:     infra.NoopLogger{},
			Signals: append(finish.DefaultSignals, syscall.SIGHUP),
		},
	}
}

func (app *App) Start(ctx context.Context, uncorsConfig *config.UncorsConfig) {
	factory := urlreplacer.NewURLReplacerFactory(uncorsConfig.Mappings)
	httpClient := infra.MakeHTTPClient(uncorsConfig.Proxy)
	cacheConfig := uncorsConfig.CacheConfig
	cacheStorage := goCache.New(cacheConfig.ExpirationTime, cacheConfig.ClearTime)

	globalHandler := handler.NewUncorsRequestHandler(
		handler.WithMappings(uncorsConfig.Mappings),
		handler.WithLogger(ui.MockLogger),
		handler.WithCacheMiddlewareFactory(func(globs config.CacheGlobs) contracts.Middleware {
			return cache.NewMiddleware(
				cache.WithLogger(ui.CacheLogger),
				cache.WithMethods(cacheConfig.Methods),
				cache.WithCacheStorage(cacheStorage),
				cache.WithGlobs(globs),
			)
		}),
		handler.WithProxyHandlerFactory(func() contracts.Handler {
			return proxy.NewProxyHandler(
				proxy.WithURLReplacerFactory(factory),
				proxy.WithHTTPClient(httpClient),
				proxy.WithLogger(ui.ProxyLogger),
			)
		}),
		handler.WithStaticHandlerFactory(func(path string, dir config.StaticDirectory) contracts.Middleware {
			return static.NewStaticMiddleware(
				static.WithFileSystem(afero.NewBasePathFs(app.fs, dir.Dir)),
				static.WithIndex(dir.Index),
				static.WithLogger(ui.StaticLogger),
				static.WithPrefix(path),
			)
		}),
		handler.WithMockHandlerFactory(func(response config.Response) contracts.Handler {
			return mock.NewMockHandler(
				mock.WithLogger(ui.MockLogger),
				mock.WithResponse(response),
				mock.WithFileSystem(app.fs),
				mock.WithAfter(time.After),
			)
		}),
	)

	uncorsServer := server.NewUncorsServer(ctx, globalHandler)

	log.Print(ui.Logo(app.version))
	log.Print("\n")
	log.Warning(ui.DisclaimerMessage)
	log.Print("\n")
	log.Info(uncorsConfig.Mappings.String())
	log.Print("\n")

	app.finisher.Add(uncorsServer)

	go func() {
		defer app.finisher.Trigger()
		log.Debugf("Starting http server on port %d", uncorsConfig.HTTPPort)
		addr := net.JoinHostPort(app.baseAddress, strconv.Itoa(uncorsConfig.HTTPPort))
		err := uncorsServer.ListenAndServe(addr)
		handleHTTPServerError("HTTP", err)
	}()

	if uncorsConfig.IsHTTPSEnabled() {
		log.Debug("Found cert file and key file. Https server will be started")
		addr := net.JoinHostPort(app.baseAddress, strconv.Itoa(uncorsConfig.HTTPSPort))
		go func() {
			defer app.finisher.Trigger()
			log.Debugf("Starting https server on port %d", uncorsConfig.HTTPSPort)
			err := uncorsServer.ListenAndServeTLS(addr, uncorsConfig.CertFile, uncorsConfig.KeyFile)
			handleHTTPServerError("HTTPS", err)
		}()
	}

	app.finisher.Wait()

	log.Info("Server was stopped")
}

func handleHTTPServerError(serverName string, err error) {
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		log.Debugf("%s server was stopped without errors", serverName)
	} else {
		panic(fmt.Errorf("%s server was stopped with error %w", serverName, err))
	}
}
