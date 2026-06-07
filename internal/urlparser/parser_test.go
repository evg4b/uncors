// nolint: lll, gosmopolitan
package urlparser_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/urlparser"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in  string
		out *url.URL
		err bool
	}{
		{in: "http://localhost", out: &url.URL{Scheme: "http", Host: "localhost"}},
		{in: "http://user.local", out: &url.URL{Scheme: "http", Host: "user.local"}},
		{in: "http://kubernetes-service", out: &url.URL{Scheme: "http", Host: "kubernetes-service"}},
		{in: hosts.Example.HTTPS(), out: &url.URL{Scheme: "https", Host: "example.com"}},
		{in: "HTTPS://example.com", out: &url.URL{Scheme: "https", Host: "example.com"}},
		{in: "ssh://example.com:22", out: &url.URL{Scheme: "ssh", Host: "example.com:22"}},
		{in: "jabber://example.com:5222", out: &url.URL{Scheme: "jabber", Host: "example.com:5222"}},
		{in: "//example.com:22", out: &url.URL{Host: "example.com:22"}},
		{in: hosts.Example.NoScheme(), out: &url.URL{Host: "example.com"}},
		{in: "localhost", out: &url.URL{Host: "localhost"}},
		{in: "LOCALHOST", out: &url.URL{Host: "localhost"}},
		{in: "localhost:80", out: &url.URL{Host: "localhost:80"}},
		{in: "localhost:8080", out: &url.URL{Host: "localhost:8080"}},
		{in: "user.local", out: &url.URL{Host: "user.local"}},
		{in: "user.local:80", out: &url.URL{Host: "user.local:80"}},
		{in: "user.local:8080", out: &url.URL{Host: "user.local:8080"}},
		{in: "kubernetes-service", out: &url.URL{Host: "kubernetes-service"}},
		{in: "kubernetes-service:80", out: &url.URL{Host: "kubernetes-service:80"}},
		{in: "kubernetes-service:8080", out: &url.URL{Host: "kubernetes-service:8080"}},
		{in: "127.0.0.1", out: &url.URL{Host: "127.0.0.1"}},
		{in: "127.0.0.1:80", out: &url.URL{Host: "127.0.0.1:80"}},
		{in: "127.0.0.1:8080", out: &url.URL{Host: "127.0.0.1:8080"}},
		{in: "[2001:db8:a0b:12f0::1]", out: &url.URL{Host: "[2001:db8:a0b:12f0::1]"}},
		{in: "[2001:db8:a0b:12f0::80]", out: &url.URL{Host: "[2001:db8:a0b:12f0::80]"}},
		{in: "http://localhost:80", out: &url.URL{Scheme: "http", Host: "localhost:80"}},
		{in: "http://localhost:8080", out: &url.URL{Scheme: "http", Host: "localhost:8080"}},
		{in: "http://x.example.io:8080", out: &url.URL{Scheme: "http", Host: "x.example.io:8080"}},
		{in: "[2001:db8:a0b:12f0::80]:80", out: &url.URL{Host: "[2001:db8:a0b:12f0::80]:80"}},
		{in: "[2001:db8:a0b:12f0::1]:8080", out: &url.URL{Host: "[2001:db8:a0b:12f0::1]:8080"}},
		{in: "example.com", out: &url.URL{Host: "example.com"}},
		{in: "1.example.com", out: &url.URL{Host: "1.example.com"}},
		{in: "1.example.io", out: &url.URL{Host: "1.example.io"}},
		{in: "subsub.sub.example.com", out: &url.URL{Host: "subsub.sub.example.com"}},
		{in: "subdomain_test.example.com", out: &url.URL{Host: "subdomain_test.example.com"}},
		{in: "user@example.com", out: &url.URL{Host: "example.com"}},
		{in: "user:passwd@example.com", out: &url.URL{Host: "example.com"}},
		{in: "https://user:passwd@subsub.sub.example.com", out: &url.URL{Scheme: "https", Host: "subsub.sub.example.com"}},
		{in: "http://user@example.com", out: &url.URL{Scheme: "http", Host: "example.com"}},
		{in: "hTTp://subSUB.sub.EXAMPLE.COM/x//////y///foo.mp3?c=z&a=x&b=y#t=20", out: &url.URL{Scheme: "http", Host: "subsub.sub.example.com", Path: "/x//////y///foo.mp3", RawQuery: "c=z&a=x&b=y", Fragment: "t=20"}},
		{in: "http://www.žluťoučký-kůň.cz/úpěl-ďábelské-ódy", err: false},
		{in: "http://www.xn--luouk-k-z2a6lsyxjlexh.cz/úpěl-ďábelské-ódy", err: false},
		{in: "http://żółć.pl/żółć.html", err: false},
		{in: "http://xn--kda4b0koi.pl/żółć.html", err: false},
		{in: "https://pressly.餐厅", err: false},
		{in: "https://pressly.组织机构", err: false},
		{in: "http://{client}.local.com", out: &url.URL{Scheme: "http", Host: "{client}.local.com"}},
		{in: "{tenant}.local.com", out: &url.URL{Host: "{tenant}.local.com"}},
		{in: "http://{region}.{tenant}.host.com", out: &url.URL{Scheme: "http", Host: "{region}.{tenant}.host.com"}},
		{in: "{tenant}.local.com:8080", out: &url.URL{Host: "{tenant}.local.com:8080"}},
		{in: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAUAAAAFCAYAAACNbyblAAAAHElEQVQI12P4//8/w38GIAXDIBKE0DHxgljNBAAO9TXL0Y4OHwAAAABJRU5ErkJggg==", err: true},
		{in: "javascript:evilFunction()", err: true},
		{in: "otherscheme:garbage", err: true},
		{in: "<funnnytag>", err: true},
		{in: hosts.Google.HTTP(), out: &url.URL{Scheme: "http", Host: "www.google.com"}},
		{in: hosts.Google.HTTPS(), out: &url.URL{Scheme: "https", Host: "www.google.com"}},
		{in: "HTTP://WWW.GOOGLE.COM", out: &url.URL{Scheme: "http", Host: "www.google.com"}},
		{in: "HTTPS://WWW.google.COM", out: &url.URL{Scheme: "https", Host: "www.google.com"}},
		{in: "http:/www.google.com", err: true},
		{in: "http:///www.google.com", err: true},
		{in: "javascript:void(0)", err: true},
		{in: "<script>", err: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.in, func(t *testing.T) {
			url, err := urlparser.Parse(testCase.in)

			if !testCase.err {
				require.NoError(t, err)
				if testCase.out != nil {
					assert.Equal(t, testCase.out.Scheme, url.Scheme)
					assert.Equal(t, testCase.out.Host, url.Host)
					if testCase.out.Path != "" {
						assert.Equal(t, testCase.out.Path, url.Path)
					}
					if testCase.out.RawQuery != "" {
						assert.Equal(t, testCase.out.RawQuery, url.RawQuery)
					}
					if testCase.out.Fragment != "" {
						assert.Equal(t, testCase.out.Fragment, url.Fragment)
					}
				}
			} else {
				require.Error(t, err)
				assert.Nil(t, url)
			}

			if testCase.out != nil && !strings.HasPrefix(strings.ToLower(testCase.in), "http://") && testCase.out.Scheme == "http" {
				url, err := urlparser.ParseWithDefaultScheme(testCase.in, "https")
				require.NoError(t, err)

				if !strings.EqualFold(url.Scheme, "https") {
					t.Errorf("%q: expected https scheme, got %q", testCase.in, url.Scheme)
				}
			}
		})
	}
}
