package rewrite

import (
	"context"
	"net/url"
	"strings"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/gorilla/mux"
)

type Middleware struct {
	rewrite config.RewritingOption
}

func NewMiddleware(options ...MiddlewareOption) *Middleware {
	return helpers.ApplyOptions(&Middleware{}, options)
}

func (m *Middleware) Wrap(next contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(writer contracts.ResponseWriter, request *contracts.Request) {
		m.rewriteURL(request)
		next.ServeHTTP(writer, m.rewriteRequest(request))
	})
}

func (m *Middleware) rewriteURL(request *contracts.Request) {
	clonedURL := &url.URL{Path: replace(m.rewrite.To, mux.Vars(request))}
	request.URL = request.URL.ResolveReference(clonedURL)
}

func (m *Middleware) rewriteRequest(request *contracts.Request) *contracts.Request {
	if m.rewrite.Host == "" {
		return request
	}

	return request.WithContext(
		context.WithValue(request.Context(), rewriteHostKey, m.rewrite.Host),
	)
}

func replace(s string, data map[string]string) string {
	for key, value := range data {
		s = strings.ReplaceAll(s, "{"+key+"}", value)
	}

	return s
}
