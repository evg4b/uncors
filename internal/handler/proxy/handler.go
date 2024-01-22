package proxy

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/urlreplacer"
)

type Handler struct {
	replacers urlreplacer.ReplacerFactory
	http      contracts.HTTPClient
	logger    contracts.Logger
}

func NewProxyHandler(options ...HandlerOption) *Handler {
	middleware := helpers.ApplyOptions(&Handler{}, options)

	helpers.AssertIsDefined(middleware.replacers, "ProxyHandler: ReplacerFactory is not configured")
	helpers.AssertIsDefined(middleware.logger, "ProxyHandler: Logger is not configured")
	helpers.AssertIsDefined(middleware.http, "ProxyHandler: Http client is not configured")

	return middleware
}

func (h *Handler) ServeHTTP(response contracts.ResponseWriter, request *contracts.Request) {
	if err := h.handle(response, request); err != nil {
		infra.HTTPError(response, err)
	}
}

func (h *Handler) handle(resp http.ResponseWriter, req *http.Request) error {
	if strings.EqualFold(req.Method, http.MethodOptions) {
		return h.makeOptionsResponse(resp, req)
	}

	targetReplacer, sourceReplacer, err := h.replacers.Make(req.URL)
	if err != nil {
		return fmt.Errorf("failed to transform general url: %w", err)
	}

	originalRequest, err := h.makeOriginalRequest(req, targetReplacer)
	if err != nil {
		return fmt.Errorf("failed to create reuest to original source: %w", err)
	}

	originalResponse, err := h.executeQuery(originalRequest)
	if err != nil {
		return err
	}

	defer helpers.CloseSafe(originalResponse.Body)

	err = h.makeUncorsResponse(originalResponse, resp, sourceReplacer)
	if err != nil {
		return fmt.Errorf("failed to make uncors response: %w", err)
	}

	return nil
}

func (h *Handler) executeQuery(request *http.Request) (*http.Response, error) {
	originalResponse, err := h.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to do reuest: %w", err)
	}
	//h.logger.PrintResponse(originalResponse.Request, originalResponse.StatusCode)

	return originalResponse, nil
}

func copyCookiesToSource(target *http.Response, replacer *urlreplacer.Replacer, source http.ResponseWriter) {
	for _, cookie := range target.Cookies() {
		cookie.Secure = replacer.IsTargetSecure()
		cookie.Domain = replacer.ReplaceSoft(cookie.Domain)
		http.SetCookie(source, cookie)
	}
}

func copyCookiesToTarget(source *http.Request, replacer *urlreplacer.Replacer, target *http.Request) {
	for _, cookie := range source.Cookies() {
		cookie.Secure = replacer.IsTargetSecure()
		cookie.Domain = replacer.ReplaceSoft(cookie.Domain)
		target.AddCookie(cookie)
	}
}
