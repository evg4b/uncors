package proxy

import (
	"fmt"
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/internal/urlreplacer"
)

type Handler struct {
	replacers     urlreplacer.ReplacerFactory
	http          contracts.HTTPClient
	proxyLogger   contracts.Logger
	rewriteLogger contracts.Logger
}

func NewProxyHandler(options ...HandlerOption) *Handler {
	middleware := helpers.ApplyOptions(&Handler{}, options)

	helpers.AssertIsDefined(middleware.replacers, "ProxyHandler: ReplacerFactory is not configured")
	helpers.AssertIsDefined(middleware.proxyLogger, "ProxyHandler: Logger is not configured")
	helpers.AssertIsDefined(middleware.rewriteLogger, "ProxyHandler: Logger is not configured")
	helpers.AssertIsDefined(middleware.http, "ProxyHandler: Http client is not configured")

	return middleware
}

func (h *Handler) ServeHTTP(response contracts.ResponseWriter, request *contracts.Request) {
	if err := h.handle(response, request); err != nil {
		infra.HTTPError(response, err)
	}
}

func (h *Handler) handle(resp http.ResponseWriter, req *http.Request) error {
	targetReplacer, sourceReplacer, err := h.careteReplacers(req)
	if err != nil {
		return err
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

func (h *Handler) careteReplacers(req *http.Request) (*urlreplacer.Replacer, *urlreplacer.Replacer, error) {
	rewriteHost, err := rewrite.GetRewriteHost(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get rewrite host: %w", err)
	}

	if rewriteHost == "" {
		targetReplacer, sourceReplacer, err := h.replacers.Make(req.URL)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to transform general url: %w", err)
		}

		return targetReplacer, sourceReplacer, nil
	}

	target, err := urlreplacer.NewReplacer(req.URL.Host, rewriteHost)
	if err != nil {
		return nil, nil, err
	}

	source, err := urlreplacer.NewReplacer(rewriteHost, req.URL.Host)
	if err != nil {
		return nil, nil, err
	}

	return target, source, nil
}

func (h *Handler) executeQuery(request *http.Request) (*http.Response, error) {
	originalResponse, err := h.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to do reuest: %w", err)
	}
	tui.PrintResponse(h.logger(request), originalResponse.Request, originalResponse.StatusCode)

	return originalResponse, nil
}

func (h *Handler) logger(requst *http.Request) contracts.Logger {
	if rewrite.IsRewriteRequest(requst) {
		return h.rewriteLogger
	}

	return h.proxyLogger
}
