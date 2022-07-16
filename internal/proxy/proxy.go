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
	replacerFactory *urlreplacer.URLReplacerFactory
	http            http.Client
}

func NewProxyMiddleware(options ...proxyMiddlewareOption) *ProxyMiddleware {
	middleware := &ProxyMiddleware{}

	for _, option := range options {
		option(middleware)
	}

	return middleware
}

func (pm *ProxyMiddleware) Wrap(next infrastructure.HandlerFunc) infrastructure.HandlerFunc {
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

		url, _ := replacer.ToTarget(req.URL.String())
		targetReq, err := http.NewRequestWithContext(req.Context(), req.Method, url, req.Body)
		if err != nil {
			pterm.Error.Println(err)
			resp.WriteHeader(http.StatusInternalServerError)
			_, wErr := resp.Write([]byte(err.Error()))
			if wErr != nil {
				panic(wErr)
			}

			return fmt.Errorf("failed to make requst to original server: %w", err)
		}

		err = copyHeaders(req.Header, targetReq.Header, map[string]func(string) (string, error){
			"origin":  replacer.ToTarget,
			"referer": replacer.ToTarget,
		})

		if err != nil {
			return err
		}

		for _, cookie := range req.Cookies() {
			cookie.Secure = true
			targetReq.AddCookie(cookie)
		}

		targetResp, err := pm.http.Do(targetReq)
		if err != nil {
			return fmt.Errorf("failed to recive response from original server: %w", err)
		}

		defer targetResp.Body.Close()

		for _, cookie := range targetResp.Cookies() {
			cookie.Secure = false
			http.SetCookie(resp, cookie)
		}

		header := resp.Header()
		err = copyHeaders(targetResp.Header, header, map[string]func(string) (string, error){
			"location": replacer.ToSource,
		})

		if err != nil {
			return err
		}

		header.Set("Access-Control-Allow-Origin", "*")
		header.Set("Access-Control-Allow-Credentials", "true")
		header.Set("Access-Control-Allow-Methods", "GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS")

		resp.WriteHeader(targetResp.StatusCode)

		_, err = io.Copy(resp, targetResp.Body)
		if err != nil {
			return fmt.Errorf("failed to copy body to response: %w", err)
		}

		proxyWriter.Println(responseprinter.Printresponse(targetResp))

		return nil
	}
}
