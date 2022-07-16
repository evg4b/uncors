package proxy

import (
	"io"
	"net/http"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/evg4b/uncors/internal/responceprinter"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/pterm/pterm"
)

type ProxyMiddleware struct {
	replacerFactory *urlreplacer.UrlReplacerFactory
	http            http.Client
}

func NewProxyHandlingMiddleware(options ...proxyMiddlewareOption) *ProxyMiddleware {
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
			Text:  "PROXY",
		},
	}

	return func(w http.ResponseWriter, r *http.Request) error {
		replacer, err := pm.replacerFactory.Make(r.URL)
		if err != nil {
			return err
		}

		if r.Method == "OPTIONS" {
			return pm.hadnleOptionsRequest(w, r)
		}

		url, _ := replacer.ToTarget(r.URL.String())
		targetReq, err := http.NewRequest(r.Method, url, r.Body)
		if err != nil {
			pterm.Error.Println(err)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return err
		}

		err = copyHeaders(r.Header, targetReq.Header, map[string]func(string) (string, error){
			"origin":  replacer.ToTarget,
			"referer": replacer.ToTarget,
		})

		if err != nil {
			return err
		}

		for _, cookie := range r.Cookies() {
			cookie.Secure = true
			targetReq.AddCookie(cookie)
		}

		targetResp, err := pm.http.Do(targetReq)
		if err != nil {
			return err
		}

		for _, cookie := range targetResp.Cookies() {
			cookie.Secure = false
			http.SetCookie(w, cookie)
		}

		header := w.Header()
		err = copyHeaders(targetResp.Header, header, map[string]func(string) (string, error){
			"location": replacer.ToSource,
		})

		if err != nil {
			return err
		}

		header.Set("Access-Control-Allow-Origin", "*")
		header.Set("Access-Control-Allow-Credentials", "true")
		header.Set("Access-Control-Allow-Methods", "GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS")

		w.WriteHeader(targetResp.StatusCode)

		_, err = io.Copy(w, targetResp.Body)
		if err != nil {
			return err
		}

		proxyWriter.Println(responceprinter.PrintResponce(targetResp))

		return nil
	}
}
