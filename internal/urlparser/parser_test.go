// nolint: lll, gosmopolitan
package urlparser_test

import (
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
		out string
		err bool
	}{
		// Error out on missing host:
		{in: "", err: true},
		{in: "/", err: true},
		{in: "//", err: true},

		// // Test schemes:
		{in: "http://example.com", out: "http://example.com"},
		{in: "HTTP://x.example.com", out: "http://x.example.com"},
		{in: "http://localhost", out: "http://localhost"},
		{in: "http://user.local", out: "http://user.local"},
		{in: "http://kubernetes-service", out: "http://kubernetes-service"},
		{in: hosts.Example.HTTPS(), out: hosts.Example.HTTPS()},
		{in: "HTTPS://example.com", out: "https://example.com"},
		{in: "ssh://example.com:22", out: "ssh://example.com:22"},
		{in: "jabber://example.com:5222", out: "jabber://example.com:5222"},
		{in: "//example.com:22", out: "//example.com:22"},
		{in: hosts.Example.NoScheme(), out: hosts.Example.NoScheme()},

		// // Empty scheme
		{in: "localhost", out: "//localhost"},
		{in: "LOCALHOST", out: "//localhost"},
		{in: "localhost:80", out: "//localhost:80"},
		{in: "localhost:8080", out: "//localhost:8080"},
		{in: "user.local", out: "//user.local"},
		{in: "user.local:80", out: "//user.local:80"},
		{in: "user.local:8080", out: "//user.local:8080"},
		{in: "kubernetes-service", out: "//kubernetes-service"},
		{in: "kubernetes-service:80", out: "//kubernetes-service:80"},
		{in: "kubernetes-service:8080", out: "//kubernetes-service:8080"},
		{in: "127.0.0.1", out: "//127.0.0.1"},
		{in: "127.0.0.1:80", out: "//127.0.0.1:80"},
		{in: "127.0.0.1:8080", out: "//127.0.0.1:8080"},
		{in: "[2001:db8:a0b:12f0::1]", out: "//[2001:db8:a0b:12f0::1]"},
		{in: "[2001:db8:a0b:12f0::80]", out: "//[2001:db8:a0b:12f0::80]"},

		// // Keep the port even on matching scheme:
		{in: "http://localhost:80", out: "http://localhost:80"},
		{in: "http://localhost:8080", out: "http://localhost:8080"},
		{in: "http://x.example.io:8080", out: "http://x.example.io:8080"},
		{in: "[2001:db8:a0b:12f0::80]:80", out: "//[2001:db8:a0b:12f0::80]:80"},
		{in: "[2001:db8:a0b:12f0::1]:8080", out: "//[2001:db8:a0b:12f0::1]:8080"},

		// // Test domains, subdomains etc.:
		{in: "example.com", out: "//example.com"},
		{in: "1.example.com", out: "//1.example.com"},
		{in: "1.example.io", out: "//1.example.io"},
		{in: "subsub.sub.example.com", out: "//subsub.sub.example.com"},
		{in: "subdomain_test.example.com", out: "//subdomain_test.example.com"},

		// // Test userinfo:
		{in: "user@example.com", out: "//user@example.com"},
		{in: "user:passwd@example.com", out: "//user:passwd@example.com"},
		{in: "https://user:passwd@subsub.sub.example.com", out: "https://user:passwd@subsub.sub.example.com"},
		{in: "http://user@example.com", out: "http://user@example.com"},

		// // Lowercase scheme and host by default. Let net/url normalize URL by default:
		{in: "hTTp://subSUB.sub.EXAMPLE.COM/x//////y///foo.mp3?c=z&a=x&b=y#t=20", out: "http://subsub.sub.example.com/x//////y///foo.mp3?c=z&a=x&b=y#t=20"},

		// // IDNA Punycode domains.
		{in: "http://www.žluťoučký-kůň.cz/úpěl-ďábelské-ódy", out: "http://www.%C5%BElu%C5%A5ou%C4%8Dk%C3%BD-k%C5%AF%C5%88.cz/%C3%BAp%C4%9Bl-%C4%8F%C3%A1belsk%C3%A9-%C3%B3dy"},
		{in: "http://www.xn--luouk-k-z2a6lsyxjlexh.cz/úpěl-ďábelské-ódy", out: "http://www.xn--luouk-k-z2a6lsyxjlexh.cz/%C3%BAp%C4%9Bl-%C4%8F%C3%A1belsk%C3%A9-%C3%B3dy"},
		{in: "http://żółć.pl/żółć.html", out: "http://%C5%BC%C3%B3%C5%82%C4%87.pl/%C5%BC%C3%B3%C5%82%C4%87.html"},
		{in: "http://xn--kda4b0koi.pl/żółć.html", out: "http://xn--kda4b0koi.pl/%C5%BC%C3%B3%C5%82%C4%87.html"},

		// // IANA TLDs.
		{in: "https://pressly.餐厅", out: "https://pressly.%E9%A4%90%E5%8E%85"},
		{in: "https://pressly.组织机构", out: "https://pressly.%E7%BB%84%E7%BB%87%E6%9C%BA%E6%9E%84"},

		// // Some obviously wrong data:
		{in: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAUAAAAFCAYAAACNbyblAAAAHElEQVQI12P4//8/w38GIAXDIBKE0DHxgljNBAAO9TXL0Y4OHwAAAABJRU5ErkJggg==", err: true},
		{in: "javascript:evilFunction()", err: true},
		{in: "otherscheme:garbage", err: true},
		{in: "<funnnytag>", err: true},

		{in: "http://www.google.com", out: "http://www.google.com"},
		{in: "https://www.google.com", out: "https://www.google.com"},
		{in: "HTTP://WWW.GOOGLE.COM", out: "http://www.google.com"},
		{in: "HTTPS://WWW.google.COM", out: "https://www.google.com"},
		{in: "http:/www.google.com", err: true},
		{in: "http:///www.google.com", err: true},
		{in: "javascript:void(0)", err: true},
		{in: "<script>", err: true},
		{in: "http:/www.google.com", err: true},
	}

	for _, testCase := range tests {
		url, err := urlparser.Parse(testCase.in)

		if !testCase.err {
			require.NoError(t, err)
		}
		if testCase.err {
			require.Error(t, err)
		}

		if testCase.out != "" {
			assert.Equal(t, testCase.out, url.String())

			// If the above defaulted to HTTP, let's test HTTPS too.
			if !strings.HasPrefix(strings.ToLower(testCase.in), "http://") && strings.HasPrefix(testCase.out, "http://") {
				url, err := urlparser.ParseWithDefaultScheme(testCase.in, "https")
				require.NoError(t, err)

				if !strings.HasPrefix(url.String(), "https://") {
					t.Errorf("%q: expected %q with https:// prefix, got %q", testCase.in, testCase.out, url.String())
				}
			}
		}
	}
}
