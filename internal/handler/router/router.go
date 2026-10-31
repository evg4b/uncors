package router

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/infra"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

func setDefaultHandler(router *mux.Router, handler contracts.Handler) {
	httpHandler := infra.CastToHTTPHandler(handler)
	router.NotFoundHandler = httpHandler
	router.MethodNotAllowedHandler = httpHandler
}

type Factory interface {
	StaticMiddleware(path string, dir config.StaticDirectory) contracts.Middleware
	RewriteMiddleware(rewriting *config.RewritingOption) contracts.Middleware
	HARMiddleware(harConfig *config.HARConfig) contracts.Middleware
	ScriptHandler(scriptConfig *config.Script) contracts.Handler
	OptionsMiddleware(cfg config.OptionsHandling) contracts.Middleware
	MockHandler(response *config.Response) contracts.Handler
}

type Router struct {
	*mux.Router

	defaultHandler contracts.Handler
	factory        Factory

	cacheMiddlewareFactory CacheMiddlewareFactory
}

func NewRouter(mappings config.Mappings, options ...Option) *Router {
	instance := Router{
		Router: mux.NewRouter().SkipClean(true),
	}

	for _, option := range options {
		option(&instance)
	}

	for _, mapping := range mappings {
		instance.registerMapping(mapping)
	}

	setDefaultHandler(instance.Router, infra.HandlerFunc(func(_ contracts.ResponseWriter, request *http.Request) error {
		if span := trace.SpanFromContext(request.Context()); span.IsRecording() {
			span.AddEvent("uncors.host_not_mapped", trace.WithAttributes(semconv.ServerAddress(request.Host)))
		}

		return fmt.Errorf("%w: %s", infra.ErrHostNotMapped, request.Host)
	}))

	return &instance
}

func (r *Router) registerMapping(mapping config.Mapping) {
	router := r.Router.Host(mapping.From.Hostname).
		Subrouter()

	defaultHandler := r.prepareDefaultHandler(mapping)

	for _, staticDir := range mapping.Statics {
		middleware := r.factory.StaticMiddleware(staticDir.Path, staticDir)
		handler := traced(infra.Wrap(middleware, defaultHandler), slices.Concat(
			mappingAttributes(mapping, routeStatic),
			[]attribute.KeyValue{
				semconv.HTTPRoute(staticDir.Path),
				attribute.String("uncors.static.dir", staticDir.Dir),
				attribute.String("uncors.static.index", staticDir.Index),
			},
		)...)
		registerPrefixHandler(router, staticDir.Path, handler)
	}

	registerMatchedRoutes(mapping.Mocks,
		func(m *config.Mock) *config.RequestMatcher { return &m.Matcher },
		func(index int, def *config.Mock) {
			mock := r.withPreflight(mapping, def.Matcher, r.factory.MockHandler(&def.Response))
			handler := traced(checkOrigin(mapping.AllowedOrigins, mock), slices.Concat(
				mappingAttributes(mapping, routeMock),
				matcherAttributes(def.Matcher, index),
			)...)
			registerRoute(createRoute(router, def.Matcher), handler)
		})

	registerMatchedRoutes(mapping.Scripts,
		func(s *config.Script) *config.RequestMatcher { return &s.Matcher },
		func(index int, def *config.Script) {
			script := r.withPreflight(mapping, def.Matcher, r.factory.ScriptHandler(def))
			handler := traced(checkOrigin(mapping.AllowedOrigins, script), slices.Concat(
				mappingAttributes(mapping, routeScript),
				matcherAttributes(def.Matcher, index),
			)...)
			registerRoute(createRoute(router, def.Matcher), handler)
		})

	for index, rewrite := range mapping.Rewrites {
		handler := traced(infra.Wrap(r.factory.RewriteMiddleware(&rewrite), defaultHandler), slices.Concat(
			mappingAttributes(mapping, routeRewrite),
			[]attribute.KeyValue{
				semconv.HTTPRoute(rewrite.From),
				attribute.Int("uncors.route.index", index),
			},
		)...)

		registerPathHandler(router, rewrite.From, handler)
	}

	setDefaultHandler(router, traced(defaultHandler, mappingAttributes(mapping, routeDefault)...))
}

func (r *Router) prepareDefaultHandler(mapping config.Mapping) contracts.Handler {
	defaultHandler := r.withPreflight(mapping, config.RequestMatcher{}, r.defaultHandler)

	if len(mapping.Cache) > 0 || mapping.HAR.Enabled() {
		defaultHandler = bypassOnUpgrade(r.recordedHandler(mapping, defaultHandler), defaultHandler)
	}

	return checkOrigin(mapping.AllowedOrigins, defaultHandler)
}

func (r *Router) withPreflight(
	mapping config.Mapping,
	matcher config.RequestMatcher,
	handler contracts.Handler,
) contracts.Handler {
	if mapping.OptionsHandling.Disabled || matcher.Method == http.MethodOptions {
		return handler
	}

	return infra.Wrap(r.factory.OptionsMiddleware(mapping.OptionsHandling), handler)
}

func (r *Router) recordedHandler(mapping config.Mapping, handler contracts.Handler) contracts.Handler {
	if len(mapping.Cache) > 0 {
		handler = infra.Wrap(r.cacheMiddlewareFactory(mapping.Cache), handler)
	}

	if mapping.HAR.Enabled() {
		handler = infra.Wrap(r.factory.HARMiddleware(&mapping.HAR), handler)
	}

	return handler
}

func bypassOnUpgrade(handler, upgradeHandler contracts.Handler) contracts.Handler {
	return infra.HandlerFunc(func(writer contracts.ResponseWriter, request *contracts.Request) error {
		if infra.IsUpgrade(request.Header) {
			return upgradeHandler.ServeHTTP(writer, request)
		}

		return handler.ServeHTTP(writer, request)
	})
}
