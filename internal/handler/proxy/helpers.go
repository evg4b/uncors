package proxy

import (
	"net"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/evg4b/uncors/internal/headers"
	"github.com/evg4b/uncors/internal/urlreplacer"
)

type modificationsMap = map[string]func(string) string

func noop(s string) string { return s }

func copyHeaders(source, dest http.Header, modifications modificationsMap) {
	for key, values := range source {
		if key == headers.SetCookie || key == headers.ContentLength {
			continue
		}

		modificationFunc, ok := modifications[http.CanonicalHeaderKey(key)]
		if !ok {
			modificationFunc = noop
		}

		for _, value := range values {
			dest.Add(key, modificationFunc(value))
		}
	}
}

const connectionHeader = "Connection"

var hopByHopHeaders = []string{
	connectionHeader,
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

func removeHopByHopHeaders(header http.Header) {
	for _, value := range header.Values(connectionHeader) {
		for token := range strings.SplitSeq(value, ",") {
			if name := textproto.TrimString(token); name != "" {
				header.Del(name)
			}
		}
	}

	for _, name := range hopByHopHeaders {
		header.Del(name)
	}
}

func hasToken(values []string, token string) bool {
	for _, value := range values {
		for item := range strings.SplitSeq(value, ",") {
			if strings.EqualFold(textproto.TrimString(item), token) {
				return true
			}
		}
	}

	return false
}

func copyCookiesToSource(target *http.Response, replacer *urlreplacer.Replacer, source http.ResponseWriter) {
	for _, cookie := range target.Cookies() {
		cookie.Secure = replacer.IsTargetSecure()

		if !cookie.Secure && cookie.SameSite == http.SameSiteNoneMode {
			cookie.SameSite = http.SameSiteLaxMode
		}

		cookie.Domain = localDomain(replacer, cookie.Domain)

		http.SetCookie(source, cookie)
	}
}

func localDomain(replacer *urlreplacer.Replacer, domain string) string {
	local, err := replacer.Replace(domain)
	if err != nil {
		return ""
	}

	host, _, err := net.SplitHostPort(local)
	if err != nil {
		return local
	}

	return host
}
