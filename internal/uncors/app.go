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

// portServer holds a server and its listener for a specific port.
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
	for _, ps := range app.servers {
		if err := ps.server.Close(); err != nil && firstErr == nil {
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

// GetListenerAddr returns the listener address for a given port.
func (app *App) GetListenerAddr(port int) net.Addr {
	app.serversMutex.RLock()
	defer app.serversMutex.RUnlock()

	ps, exists := app.servers[port]
	if !exists || ps == nil {
		return nil
	}

	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if ps.listener == nil {
		return nil
	}

	return ps.listener.Addr()
}

// HTTPAddr returns the first HTTP listener address (for backward compatibility).
func (app *App) HTTPAddr() net.Addr {
	app.serversMutex.RLock()
	defer app.serversMutex.RUnlock()

	for _, ps := range app.servers {
		if ps.scheme == "http" && ps.listener != nil {
			return ps.listener.Addr()
		}
	}

	return nil
}

// HTTPSAddr returns the first HTTPS listener address (for backward compatibility).
func (app *App) HTTPSAddr() net.Addr {
	app.serversMutex.RLock()
	defer app.serversMutex.RUnlock()

	for _, ps := range app.servers {
		if ps.scheme == "https" && ps.listener != nil {
			return ps.listener.Addr()
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
		ps := &portServer{
			port:   group.Port,
			scheme: group.Scheme,
			mutex:  &sync.Mutex{},
		}

		// Create server with handler for this port's mappings
		ps.server = app.createServerForPort(ctx, uncorsConfig, group.Mappings)

		app.serversMutex.Lock()
		app.servers[group.Port] = ps
		app.serversMutex.Unlock()

		// Start listener for this port
		app.waitGroup.Add(1)
		go app.startListener(ctx, ps, uncorsConfig)
	}
}

func (app *App) startListener(ctx context.Context, ps *portServer, uncorsConfig *config.UncorsConfig) {
	defer app.waitGroup.Done()
	defer ps.mutex.Unlock()

	ps.mutex.Lock()

	addr := net.JoinHostPort(baseAddress, strconv.Itoa(ps.port))
	serverName := fmt.Sprintf("%s:%d", strings.ToUpper(ps.scheme), ps.port)

	log.Debugf("Starting %s server on port %d", ps.scheme, ps.port)

	var err error
	if ps.scheme == "https" {
		if !uncorsConfig.IsHTTPSEnabled() {
			log.Warnf("HTTPS mapping on port %d found but no cert/key configured, skipping", ps.port)

			return
		}
		err = app.listenAndServeTLSForPort(ps, addr, uncorsConfig.CertFile, uncorsConfig.KeyFile)
	} else {
		err = app.listenAndServeForPort(ps, addr)
	}

	handleHTTPServerError(serverName, err)
}

func (app *App) createServerForPort(ctx context.Context, uncorsConfig *config.UncorsConfig, mappings config.Mappings) *http.Server {
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
