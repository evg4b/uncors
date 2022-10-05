package proxy

import (
	"fmt"
	"io"
	"net/http"

	"github.com/evg4b/uncors/internal/infrastructure"
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

func (pm *ProxyMiddleware) Wrap(_ infrastructure.HandlerFunc) infrastructure.HandlerFunc {
	proxyWriter := pterm.PrefixPrinter{
		MessageStyle: &pterm.ThemeDefault.InfoMessageStyle,
		Prefix: pterm.Prefix{
			Style: &pterm.Style{pterm.FgBlack, pterm.BgLightBlue},
			Text:  " PROXY ",
		},
	}

	return func(resp http.ResponseWriter, req *http.Request) error {
		replacer, err := pm.replacerFactory.Make(req.URL)
		if err != nil {
			return fmt.Errorf("failed to transform general url: %w", err)
		}

		originalReq, err := pm.makeOriginalRequest(req, replacer)
		if err != nil {
			return fmt.Errorf("failed to recive response from original server: %w", err)
		}

		originalResp, err := pm.http.Do(originalReq)
		if err != nil {
			return fmt.Errorf("failed to recive response from original server: %w", err)
		}

		defer originalResp.Body.Close()

		err = pm.makeUncorsResponse(originalResp, resp, replacer)
		if err != nil {
			return fmt.Errorf("failed to recive response from original server: %w", err)
		}

		proxyWriter.Println(responseprinter.PrintResponse(originalResp))

		return nil
	}
}

// nolint: unparam
func copyCookiesToSource(target *http.Response, replacer *urlreplacer.Replacer, source http.ResponseWriter) error {
	for _, cookie := range target.Cookies() {
		cookie.Secure = replacer.IsSourceSecure()
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
