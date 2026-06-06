package handler

import (
	"errors"
	"net/http"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/gorilla/mux"
)

var errHostNotMapped = errors.New("host not mapped")

func setDefaultHandler(router *mux.Router, handler contracts.Handler) {
	httpHandler := contracts.CastToHTTPHandler(handler)
	router.NotFoundHandler = httpHandler
	router.MethodNotAllowedHandler = httpHandler
}

type Router struct {
	*mux.Router

	defaultHandler contracts.Handler

	cacheMiddlewareFactory   CacheMiddlewareFactory
	staticMiddlewareFactory  StaticMiddlewareFactory
	mockHandlerFactory       MockHandlerFactory
	scriptHandlerFactory     ScriptHandlerFactory
	rewriteMiddlewareFactory RewriteMiddlewareFactory
	optionsMiddlewareFactory OptionsMiddlewareFactory
	harMiddlewareFactory     HARMiddlewareFactory
}

func NewRouter(mappings config.Mappings, options ...RouterOption) (*Router, error) {
	instance := Router{
		Router: mux.NewRouter(),
	}

	for _, option := range options {
		option(&instance)
	}

	for _, mapping := range mappings {
		err := instance.registerMapping(mapping)
		if err != nil {
			return nil, err
		}
	}

	setDefaultHandler(instance.Router, contracts.HandlerFunc(func(_ contracts.ResponseWriter, _ *http.Request) error {
		// instance.output.Errorf("Host %s://%s is not mapped", r.URL.Scheme, r.URL.Host)
		// log.Printf("Host %s://%s is not mapped", r.URL.Scheme, r.URL.Host) // nolint: gosec
		return errHostNotMapped
	}))

	return &instance, nil
}

func (r *Router) registerMapping(mapping config.Mapping) error {
	host, _, err := mapping.GetFromHostPort()
	if err != nil {
		return err
	}

	router := r.Router.Host(replaceWildcards(host)).
		Subrouter()

	defaultHandler := r.prepareDefaultHandler(mapping)

	for _, staticDir := range mapping.Statics {
		handler := r.staticMiddlewareFactory(staticDir.Path, staticDir).
			Wrap(defaultHandler)

		registerPrefixHandler(router, staticDir.Path, handler)
	}

	registerMatchedRoutes(mapping.Mocks,
		func(m *config.Mock) *config.RequestMatcher { return &m.Matcher },
		func(def *config.Mock) {
			registerRoute(createRoute(router, def.Matcher), r.mockHandlerFactory(def.Response))
		})

	registerMatchedRoutes(mapping.Scripts,
		func(s *config.Script) *config.RequestMatcher { return &s.Matcher },
		func(def *config.Script) {
			registerRoute(createRoute(router, def.Matcher), r.scriptHandlerFactory(*def))
		})

	for _, rewrite := range mapping.Rewrites {
		wrappedHandler := r.rewriteMiddlewareFactory(rewrite).
			Wrap(defaultHandler)

		registerPathHandler(router, rewrite.From, wrappedHandler)
	}

	setDefaultHandler(router, defaultHandler)

	return nil
}

func (r *Router) prepareDefaultHandler(mapping config.Mapping) contracts.Handler {
	defaultHandler := r.defaultHandler
	if !mapping.OptionsHandling.Disabled {
		defaultHandler = r.optionsMiddlewareFactory(mapping.OptionsHandling).
			Wrap(defaultHandler)
	}

	if len(mapping.Cache) > 0 {
		defaultHandler = r.cacheMiddlewareFactory(mapping.Cache).
			Wrap(defaultHandler)
	}

	if mapping.HAR.Enabled() {
		defaultHandler = r.harMiddlewareFactory(mapping.HAR).
			Wrap(defaultHandler)
	}

	return defaultHandler
}
