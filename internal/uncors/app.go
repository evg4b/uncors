// nolint: wrapcheck
package uncors

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/log"
	"github.com/spf13/afero"
	"golang.org/x/net/context"
)

type App struct {
	fs            afero.Fs
	version       string
	waitGroup     sync.WaitGroup
	httpMutex     sync.Mutex
	httpsMutex    sync.Mutex
	server        *http.Server
	shuttingDown  atomic.Bool
	httpListener  net.Listener
	httpsListener net.Listener
	cache         appCache
}

const (
	baseAddress       = "127.0.0.1"
	readHeaderTimeout = 30 * time.Second
	shutdownTimeout   = 15 * time.Second
)

func CreateApp(fs afero.Fs, version string) *App {
	return &App{fs: fs, version: version}
}

func (app *App) Start(ctx context.Context, uncorsConfig *config.UncorsConfig) {
	log.Print(Logo(app.version))
	log.Print("\n")
	log.Warning(DisclaimerMessage)
	log.Print("\n")
	log.Info(uncorsConfig.Mappings.String())
	log.Print("\n")

	app.initServer(ctx, uncorsConfig)
}

func (app *App) initServer(ctx context.Context, uncorsConfig *config.UncorsConfig) {
	app.server = app.createServer(ctx, uncorsConfig)

	app.waitGroup.Add(1)
	go func() {
		defer app.waitGroup.Done()
		defer app.httpMutex.Unlock()

		app.httpMutex.Lock()
		log.Debugf("Starting http server on port %d", uncorsConfig.HTTPPort)
		addr := net.JoinHostPort(baseAddress, strconv.Itoa(uncorsConfig.HTTPPort))
		err := app.listenAndServe(addr)
		handleHTTPServerError("HTTP", err)
	}()

	if uncorsConfig.IsHTTPSEnabled() {
		log.Debug("Found cert file and key file. Https server will be started")
		addr := net.JoinHostPort(baseAddress, strconv.Itoa(uncorsConfig.HTTPSPort))
		app.waitGroup.Add(1)
		go func() {
			defer app.waitGroup.Done()
			defer app.httpsMutex.Unlock()

			app.httpsMutex.Lock()
			log.Debugf("Starting https server on port %d", uncorsConfig.HTTPSPort)
			err := app.listenAndServeTLS(addr, uncorsConfig.CertFile, uncorsConfig.KeyFile)
			handleHTTPServerError("HTTPS", err)
		}()
	}
}

func (app *App) createServer(ctx context.Context, uncorsConfig *config.UncorsConfig) *http.Server {
	globalHandler := app.buildHandler(uncorsConfig)
	globalCtx, globalCtxCancel := context.WithCancel(ctx)
	server := &http.Server{
		BaseContext: func(listener net.Listener) context.Context {
			return globalCtx
		},
		ReadHeaderTimeout: readHeaderTimeout,
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			helpers.NormaliseRequest(request)
			globalHandler.ServeHTTP(contracts.WrapResponseWriter(writer), request)
		}),
		ErrorLog: log.StandardErrorLogAdapter(),
	}
	server.RegisterOnShutdown(globalCtxCancel)

	return server
}

func (app *App) Restart(ctx context.Context, uncorsConfig *config.UncorsConfig) {
	defer app.waitGroup.Done()
	app.waitGroup.Add(1)
	log.Print("\n")
	log.Info("Restarting server....")
	log.Print("\n")
	err := app.internalShutdown(ctx)
	if err != nil {
		panic(err) // TODO: refactor this error handling
	}

	log.Info(uncorsConfig.Mappings.String())
	log.Print("\n")
	app.initServer(ctx, uncorsConfig)
}

func (app *App) Close() error {
	return app.server.Close()
}

func (app *App) Wait() {
	app.waitGroup.Wait()
	log.Info("Server was stopped")
}

func (app *App) Shutdown(ctx context.Context) error {
	return app.internalShutdown(ctx)
}

func (app *App) HTTPAddr() net.Addr {
	return app.httpListener.Addr() // TODO: Add nil handing
}

func (app *App) HTTPSAddr() net.Addr {
	return app.httpsListener.Addr() // TODO: Add nil handing
}

func handleHTTPServerError(serverName string, err error) {
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		log.Debugf("%s server was stopped without errors", serverName)
	} else {
		panic(fmt.Errorf("%s server was stopped with error %w", serverName, err))
	}
}
