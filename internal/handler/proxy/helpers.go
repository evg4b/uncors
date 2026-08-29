package proxy

import (
	"net/http"
	"net/http/httputil"

	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/go-http-utils/headers"
)

// rewriteRequest points the outbound request at the mapped origin. Hop-by-hop
// headers, X-Forwarded-* and the request body are handled by ReverseProxy.
func rewriteRequest(request *httputil.ProxyRequest) {
	current := exchangeOf(request.In)
	if current == nil {
		return
	}

	targetURL, err := current.target.ReplaceURL(request.In.URL)
	if err != nil {
		// Serve resolved the replacers against this very URL, so a failure here
		// is not reachable; leaving the URL alone makes the transport report it.
		return
	}

	request.Out.URL = targetURL
	request.Out.Host = targetURL.Host

	request.SetXForwarded()

	rewriteHeaders(request.Out.Header, current.target, headers.Origin, headers.Referer)
}

// modifyResponse maps the upstream response back onto the source origin.
func modifyResponse(response *http.Response) error {
	current := exchangeOf(response.Request)
	if current == nil {
		return nil
	}

	if response.Header == nil {
		response.Header = http.Header{}
	}

	rewriteHeaders(response.Header, current.source, headers.Location)
	rewriteResponseCookies(response, current.source)
	infra.WriteCorsHeaders(response.Header, current.origin)

	return nil
}

func exchangeOf(request *http.Request) *exchange {
	if request == nil {
		return nil
	}

	current, _ := request.Context().Value(replacersKey{}).(*exchange)

	return current
}

// rewriteHeaders maps the URLs carried by the named headers onto the other side
// of the mapping. Values that are not URLs of the mapped host are left as they
// are.
func rewriteHeaders(header http.Header, replacer *urlreplacer.Replacer, names ...string) {
	for _, name := range names {
		values := header.Values(name)
		if len(values) == 0 {
			continue
		}

		rewritten := make([]string, 0, len(values))
		for _, value := range values {
			rewritten = append(rewritten, replacer.ReplaceSoft(value))
		}

		header.Del(name)

		for _, value := range rewritten {
			header.Add(name, value)
		}
	}
}

// rewriteResponseCookies re-issues the upstream cookies for the source host, so
// that the browser stores them against the address it actually talked to.
func rewriteResponseCookies(response *http.Response, replacer *urlreplacer.Replacer) {
	cookies := response.Cookies()
	if len(cookies) == 0 {
		return
	}

	response.Header.Del(headers.SetCookie)

	for _, cookie := range cookies {
		cookie.Secure = replacer.IsTargetSecure()
		cookie.Domain = replacer.ReplaceSoft(cookie.Domain)

		response.Header.Add(headers.SetCookie, cookie.String())
	}
}
