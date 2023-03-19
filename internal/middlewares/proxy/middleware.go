package proxy

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/pterm/pterm"
)

type Middleware struct {
	replacers URLReplacerFactory
	http      contracts.HTTPClient
	logger    contracts.Logger
}

func NewProxyMiddleware(options ...MiddlewareOption) *Middleware {
	middleware := &Middleware{}

	for _, option := range options {
		option(middleware)
	}

	helpers.AssertIsDefined(middleware.replacers, "ProxyHandler: ReplacerFactory is not configured")
	helpers.AssertIsDefined(middleware.logger, "ProxyHandler: Logger is not configured")
	helpers.AssertIsDefined(middleware.http, "ProxyHandler: Http client is not configured")

	return middleware
}

func (m *Middleware) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	updateRequest(request)

	if err := m.handle(response, request); err != nil {
		pterm.Error.Printfln("UNCORS error: %v", err)
		response.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(response, "UNCORS error:", err.Error())
	}
}

func (m *Middleware) handle(resp http.ResponseWriter, req *http.Request) error {
	if strings.EqualFold(req.Method, http.MethodOptions) {
		return m.makeOptionsResponse(resp, req)
	}

	targetReplacer, sourceReplacer, err := m.replacers.Make(req.URL)
	if err != nil {
		return fmt.Errorf("failed to transform general url: %w", err)
	}

	originalRequest, err := m.makeOriginalRequest(req, targetReplacer)
	if err != nil {
		return fmt.Errorf("failed to create reuest to original source: %w", err)
	}

	originalResponse, err := m.executeQuery(originalRequest)
	if err != nil {
		return err
	}

	defer originalResponse.Body.Close()

	err = m.makeUncorsResponse(originalResponse, resp, sourceReplacer)
	if err != nil {
		return fmt.Errorf("failed to make uncors response: %w", err)
	}

	return nil
}

func (m *Middleware) executeQuery(request *http.Request) (*http.Response, error) {
	originalResponse, err := m.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to do reuest: %w", err)
	}
	m.logger.PrintResponse(originalResponse)

	return originalResponse, nil
}

// nolint: unparam
func copyCookiesToSource(target *http.Response, replacer *urlreplacer.Replacer, source http.ResponseWriter) error {
	for _, cookie := range target.Cookies() {
		cookie.Secure = replacer.IsTargetSecure()
		// TODO: Replace domain in cookie
		http.SetCookie(source, cookie)
	}

	return nil
}

// nolint: unparam
func copyCookiesToTarget(source *http.Request, replacer *urlreplacer.Replacer, target *http.Request) error {
	for _, cookie := range source.Cookies() {
		cookie.Secure = replacer.IsTargetSecure()
		// TODO: Replace domain in cookie
		target.AddCookie(cookie)
	}

	return nil
}

func updateRequest(request *http.Request) {
	request.URL.Host = request.Host

	if request.TLS != nil {
		request.URL.Scheme = "https"
	} else {
		request.URL.Scheme = "http"
	}
}
