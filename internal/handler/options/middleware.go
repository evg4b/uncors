package options

import (
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/go-http-utils/headers"
)

type Middleware struct {
	logger  contracts.Logger
	headers map[string]string
	code    int
}

func NewMiddleware(options ...MiddlewareOption) *Middleware {
	return helpers.ApplyOptions(&Middleware{}, options)
}

func (m *Middleware) Wrap(next contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(resp contracts.ResponseWriter, req *contracts.Request) {
		if strings.EqualFold(req.Method, http.MethodOptions) {
			m.handle(resp, req)
		} else {
			next.ServeHTTP(resp, req)
		}
	})
}

const (
	defaultAllowMethods = "GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS"
	defaultMaxAge       = "86400" // 24 hours in seconds
)

// SetHeaderOrDefault sets a header to the provided value if not empty, otherwise sets it to the default value.
func SetHeaderOrDefault(header http.Header, key, value, defaultValue string) {
	if value != "" {
		header.Set(key, value)
	} else {
		header.Set(key, defaultValue)
	}
}

func (m *Middleware) handle(resp http.ResponseWriter, req *http.Request) {
	header := resp.Header()

	SetHeaderOrDefault(header, headers.AccessControlAllowOrigin, req.Header.Get(headers.Origin), "*")
	header.Set(headers.AccessControlAllowCredentials, "true")
	SetHeaderOrDefault(header, headers.AccessControlAllowHeaders, req.Header.Get(headers.AccessControlRequestHeaders), "*")
	SetHeaderOrDefault(header, headers.AccessControlAllowMethods, req.Header.Get(headers.AccessControlRequestMethod), defaultAllowMethods)
	header.Set(headers.AccessControlMaxAge, defaultMaxAge)
	header.Set(headers.AccessControlExposeHeaders, "*")

	if len(m.headers) != 0 {
		for key, value := range m.headers {
			resp.Header().Set(key, value)
		}
	}

	statucCode := helpers.NormaliseStatucCode(m.code)
	resp.WriteHeader(statucCode)

	tui.PrintResponse(m.logger, req, statucCode)
}
