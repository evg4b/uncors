package rewrite

import (
	"context"
	"net/url"
	"strings"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/gorilla/mux"
)

type Middleware struct {
	rewrite *config.RewritingOption
}

func NewMiddleware(options ...MiddlewareOption) *Middleware {
	return helpers.ApplyOptions(&Middleware{}, options)
}

func (m *Middleware) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request, next contracts.Next) error {
	m.rewriteURL(request)

	return next(writer, m.rewriteRequest(request))
}

func (m *Middleware) rewriteURL(request *contracts.Request) {
	clonedURL := &url.URL{Path: replace(m.rewrite.To, mux.Vars(request))}
	request.URL = urlt.URL_ResolveReference(request.URL, clonedURL)
}

func (m *Middleware) rewriteRequest(request *contracts.Request) *contracts.Request {
	if m.rewrite.Host == (urlt.Host{}) {
		return request
	}

	return request.WithContext(
		context.WithValue(request.Context(), RewriteHostKey, m.rewrite.Host.HostPort()),
	)
}

func replace(s string, data map[string]string) string {
	for key, value := range data {
		s = strings.ReplaceAll(s, "{"+key+"}", value)
	}

	return s
}

type MiddlewareOption = func(*Middleware)

func WithRewritingOptions(rewrite *config.RewritingOption) MiddlewareOption {
	return func(h *Middleware) {
		h.rewrite = rewrite
	}
}
