package proxy

import (
	"errors"
	"fmt"
	"net/http"
	"net/textproto"
	"net/url"
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

func copyResponseHeaders(original *http.Response, target http.ResponseWriter, replacer *urlreplacer.Replacer) error {
	err := copyCookiesToSource(original, replacer, target)
	if err != nil {
		return err
	}

	copyHeaders(original.Header, target.Header(), modificationsMap{
		headers.Location: replacer.ReplaceSoft,
	})

	return nil
}

func copyCookiesToSource(target *http.Response, replacer *urlreplacer.Replacer, source http.ResponseWriter) error {
	for _, cookie := range target.Cookies() {
		cookie.Secure = replacer.IsTargetSecure()

		if !cookie.Secure && cookie.SameSite == http.SameSiteNoneMode {
			cookie.SameSite = http.SameSiteLaxMode
		}

		domain, err := localDomain(replacer, cookie.Domain)
		if err != nil {
			return err
		}

		cookie.Domain = domain

		http.SetCookie(source, cookie)
	}

	return nil
}

func localDomain(replacer *urlreplacer.Replacer, domain string) (string, error) {
	if domain == "" {
		return "", nil
	}

	local, err := replacer.Replace(domain)
	if errors.Is(err, urlreplacer.ErrURLNotMatched) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("failed to replace cookie domain %q: %w", domain, err)
	}

	return (&url.URL{Host: local}).Hostname(), nil
}
