package proxy

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/processor"
	"github.com/evg4b/uncors/internal/responseprinter"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/pterm/pterm"
)

// nolint: revive
type ProxyMiddleware struct {
	replacerFactory URLReplacerFactory
	http            *http.Client
}

func NewProxyMiddleware(options ...MiddlewareOption) *ProxyMiddleware {
	middleware := &ProxyMiddleware{}

	for _, option := range options {
		option(middleware)
	}

	return middleware
}

func (pm *ProxyMiddleware) Wrap(_ processor.HandlerFunc) processor.HandlerFunc {
	proxyWriter := pterm.PrefixPrinter{
		MessageStyle: &pterm.ThemeDefault.InfoMessageStyle,
		Prefix: pterm.Prefix{
			Style: &pterm.Style{pterm.FgBlack, pterm.BgLightBlue},
			Text:  " PROXY ",
		},
	}

	return func(resp http.ResponseWriter, req *http.Request) error {
		if strings.EqualFold(req.Method, http.MethodOptions) {
			return makeOptionsResponse(proxyWriter, resp, req)
		}

		targetR, sourceR, err := pm.replacerFactory.Make(req.URL)
		if err != nil {
			return fmt.Errorf("failed to transform general url: %w", err)
		}

		originalReq, err := pm.makeOriginalRequest(req, targetR)
		if err != nil {
			return fmt.Errorf("failed to create reuest to original source: %w", err)
		}

		originalResp, err := pm.http.Do(originalReq)
		if err != nil {
			return fmt.Errorf("failed to do reuest: %w", err)
		}

		defer originalResp.Body.Close()

		err = pm.makeUncorsResponse(originalResp, resp, sourceR)
		if err != nil {
			return fmt.Errorf("failed to make uncors response: %w", err)
		}

		proxyWriter.Println(responseprinter.PrintResponse(originalResp))

		return nil
	}
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

func copyResponseData(header http.Header, resp http.ResponseWriter, targetResp *http.Response) error {
	header.Set("Access-Control-Allow-Origin", "*")
	header.Set("Access-Control-Allow-Credentials", "true")
	header.Set("Access-Control-Allow-Methods", "GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS")

	resp.WriteHeader(targetResp.StatusCode)

	if _, err := io.Copy(resp, targetResp.Body); err != nil {
		return fmt.Errorf("failed to copy body to response: %w", err)
	}

	return nil
}
