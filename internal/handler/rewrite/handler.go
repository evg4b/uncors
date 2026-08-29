package rewrite

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/evg4b/uncors/internal/config"
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

func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		m.rewriteURL(request)

		next.ServeHTTP(writer, m.rewriteRequest(request))
	})
}

func (m *Middleware) rewriteURL(request *http.Request) {
	clonedURL := &url.URL{Path: replace(m.rewrite.To, mux.Vars(request))}
	request.URL = urlt.URL_ResolveReference(request.URL, clonedURL)
}

func (m *Middleware) rewriteRequest(request *http.Request) *http.Request {
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
