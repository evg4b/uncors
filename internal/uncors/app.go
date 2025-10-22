package uncors

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/evg4b/uncors/internal/tui"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/spf13/afero"
	"golang.org/x/net/context"
)

type portServer struct {
	server   *http.Server
	listener net.Listener
	port     int
	scheme   string
	mutex    *sync.Mutex
}

type App struct {
	fs           afero.Fs
	version      string
	waitGroup    *sync.WaitGroup
	servers      map[int]*portServer // map port -> server
	serversMutex *sync.RWMutex
	shuttingDown *atomic.Bool
	cache        appCache
	logger       *log.Logger
}

const (
	baseAddress       = "127.0.0.1"
	readHeaderTimeout = 30 * time.Second
	shutdownTimeout   = 15 * time.Second
)

func CreateApp(fs afero.Fs, logger *log.Logger, version string) *App {
	return &App{
		fs:           fs,
		version:      version,
		waitGroup:    &sync.WaitGroup{},
		servers:      make(map[int]*portServer),
		serversMutex: &sync.RWMutex{},
		shuttingDown: &atomic.Bool{},
		logger:       logger,
	}
}

func (app *App) Start(ctx context.Context, uncorsConfig *config.UncorsConfig) {
	println(tui.Logo(app.version)) //nolint:forbidigo
	log.Print("")
	tui.PrintWarningBox(os.Stdout, DisclaimerMessage)
	log.Print("")
	tui.PrintInfoBox(os.Stdout, uncorsConfig.Mappings.String())
	log.Print("")

	app.initServer(ctx, uncorsConfig)
}

func (app *App) Restart(ctx context.Context, uncorsConfig *config.UncorsConfig) {
	defer app.waitGroup.Done()
	app.waitGroup.Add(1)
	log.Print("")
	log.Info("Restarting server....")
	log.Print("")
	err := app.internalShutdown(ctx)
	if err != nil {
		panic(err) // TODO: refactor this error handling
	}

	log.Info(uncorsConfig.Mappings.String())
	log.Print("")
	app.initServer(ctx, uncorsConfig)
}

func (app *App) Close() error {
	app.serversMutex.RLock()
	defer app.serversMutex.RUnlock()

	var firstErr error
	for _, portSrv := range app.servers {
		if err := portSrv.server.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (app *App) Wait() {
	app.waitGroup.Wait()
}

func (app *App) Shutdown(ctx context.Context) error {
	return app.internalShutdown(ctx)
}

func (app *App) GetListenerAddr(port int) net.Addr {
	app.serversMutex.RLock()
	defer app.serversMutex.RUnlock()

	portSrv, exists := app.servers[port]
	if !exists || portSrv == nil {
		return nil
	}

	portSrv.mutex.Lock()
	defer portSrv.mutex.Unlock()

	if portSrv.listener == nil {
		return nil
	}

	return portSrv.listener.Addr()
}

// HTTPAddr returns the first HTTP listener address for backward compatibility.
func (app *App) HTTPAddr() net.Addr {
	return app.getListenerAddrByScheme("http")
}

// HTTPSAddr returns the first HTTPS listener address for backward compatibility.
func (app *App) HTTPSAddr() net.Addr {
	return app.getListenerAddrByScheme("https")
}

// getListenerAddrByScheme returns the first listener address for the given scheme.
func (app *App) getListenerAddrByScheme(scheme string) net.Addr {
	app.serversMutex.RLock()
	defer app.serversMutex.RUnlock()

	for _, portSrv := range app.servers {
		if portSrv.scheme == scheme {
			portSrv.mutex.Lock()
			listener := portSrv.listener
			portSrv.mutex.Unlock()

			if listener != nil {
				return listener.Addr()
			}
		}
	}

	return nil
}

func handleHTTPServerError(serverName string, err error) {
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		log.Debugf("%s server was stopped without errors", serverName)
	} else {
		panic(fmt.Errorf("%s server was stopped with error %w", serverName, err))
	}
}

func (app *App) initServer(ctx context.Context, uncorsConfig *config.UncorsConfig) {
	app.shuttingDown.Store(false)

	// Group mappings by port
	portGroups := uncorsConfig.Mappings.GroupByPort()

	app.serversMutex.Lock()
	app.servers = make(map[int]*portServer)
	app.serversMutex.Unlock()

	// Create a server for each port group
	for _, group := range portGroups {
		portSrv := &portServer{
			port:   group.Port,
			scheme: group.Scheme,
			mutex:  &sync.Mutex{},
		}

		// Create server with handler for this port's mappings
		portSrv.server = app.createServerForPort(ctx, uncorsConfig, group.Mappings)

		app.serversMutex.Lock()
		app.servers[group.Port] = portSrv
		app.serversMutex.Unlock()

		// Start listener for this port
		app.waitGroup.Add(1)
		go app.startListener(ctx, portSrv, uncorsConfig)
	}
}

func (app *App) startListener(_ context.Context, portSrv *portServer, uncorsConfig *config.UncorsConfig) {
	defer app.waitGroup.Done()

	addr := net.JoinHostPort(baseAddress, strconv.Itoa(portSrv.port))
	serverName := fmt.Sprintf("%s:%d", strings.ToUpper(portSrv.scheme), portSrv.port)

	log.Debugf("Starting %s server on port %d", portSrv.scheme, portSrv.port)

	var err error
	if portSrv.scheme == "https" {
		if !uncorsConfig.IsHTTPSEnabled() {
			log.Warnf("HTTPS mapping on port %d found but no cert/key configured, skipping", portSrv.port)

			return
		}
		err = app.listenAndServeTLSForPort(portSrv, addr, uncorsConfig.CertFile, uncorsConfig.KeyFile)
	} else {
		err = app.listenAndServeForPort(portSrv, addr)
	}

	handleHTTPServerError(serverName, err)
}

func (app *App) createServerForPort(
	ctx context.Context,
	uncorsConfig *config.UncorsConfig,
	mappings config.Mappings,
) *http.Server {
	portHandler := app.buildHandlerForMappings(uncorsConfig, mappings)
	portCtx, portCtxCancel := context.WithCancel(ctx)
	server := &http.Server{
		BaseContext: func(_ net.Listener) context.Context {
			return portCtx
		},
		ReadHeaderTimeout: readHeaderTimeout,
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			helpers.NormaliseRequest(request)
			portHandler.ServeHTTP(contracts.WrapResponseWriter(writer), request)
		}),
		ErrorLog: log.StandardLog(),
	}
	server.RegisterOnShutdown(portCtxCancel)

	return server
}
