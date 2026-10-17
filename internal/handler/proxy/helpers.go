package proxy

import (
	"net/http"

	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/go-http-utils/headers"
)

type modificationsMap = map[string]func(string) string

func noop(s string) string { return s }

// copyHeaders copies headers except Set-Cookie, which copyCookiesToSource
// rewrites, and Content-Length, which net/http sets from the body. The Cookie
// header is copied unchanged.
func copyHeaders(source, dest http.Header, modifications modificationsMap) {
	for key, values := range source {
		if key == headers.SetCookie || key == headers.ContentLength {
			continue
		}

		modificationFunc, ok := modifications[headers.Normalize(key)]
		if !ok {
			modificationFunc = noop
		}

		for _, value := range values {
			dest.Add(key, modificationFunc(value))
		}
	}
}

func copyCookiesToSource(target *http.Response, replacer *urlreplacer.Replacer, source http.ResponseWriter) {
	for _, cookie := range target.Cookies() {
		cookie.Secure = replacer.IsTargetSecure()
		cookie.Domain = replacer.ReplaceSoft(cookie.Domain)
		http.SetCookie(source, cookie)
	}
}
