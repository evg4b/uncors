package proxy

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/pterm/pterm"
)

type Handler struct {
	replacerFactory URLReplacerFactory
	http            *http.Client
	logger          contracts.Logger
}

func NewProxyHandler(options ...HandlerOption) *Handler {
	handler := &Handler{}

	for _, option := range options {
		option(handler)
	}

	return handler
}

func (handler *Handler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	updateRequest(request)

	if err := handler.handle(response, request); err != nil {
		pterm.Error.Printfln("UNCORS error: %v", err)
		response.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(response, "UNCORS error:", err.Error())
	}
}

func (handler *Handler) handle(resp http.ResponseWriter, req *http.Request) error {
	if strings.EqualFold(req.Method, http.MethodOptions) {
		return handler.makeOptionsResponse(resp, req)
	}

	targetR, sourceR, err := handler.replacerFactory.Make(req.URL)
	if err != nil {
		return fmt.Errorf("failed to transform general url: %w", err)
	}

	originalReq, err := handler.makeOriginalRequest(req, targetR)
	if err != nil {
		return fmt.Errorf("failed to create reuest to original source: %w", err)
	}

	originalResp, err := handler.http.Do(originalReq)
	if err != nil {
		return fmt.Errorf("failed to do reuest: %w", err)
	}

	defer originalResp.Body.Close()

	err = handler.makeUncorsResponse(originalResp, resp, sourceR)
	if err != nil {
		return fmt.Errorf("failed to make uncors response: %w", err)
	}

	handler.logger.PrintResponse(originalResp)

	return nil
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

func copyResponseData(resp http.ResponseWriter, targetResp *http.Response) error {
	resp.WriteHeader(targetResp.StatusCode)

	if _, err := io.Copy(resp, targetResp.Body); err != nil {
		return fmt.Errorf("failed to copy body to response: %w", err)
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
