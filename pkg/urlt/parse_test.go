// nolint: lll, gosmopolitan, gosec
package urlt_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParse exercises the opinionated, host-biased urlt.Parse migrated from the
// former internal/urlparser package.
func TestParse(t *testing.T) { // NOSONAR
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
		{in: "user.local:8080", out: &url.URL{Host: "user.local:8080"}},
		{in: "kubernetes-service", out: &url.URL{Host: "kubernetes-service"}},
		{in: "kubernetes-service:8080", out: &url.URL{Host: "kubernetes-service:8080"}},
		{in: "127.0.0.1", out: &url.URL{Host: "127.0.0.1"}},
		{in: "127.0.0.1:8080", out: &url.URL{Host: "127.0.0.1:8080"}},
		{in: "[2001:db8:a0b:12f0::1]", out: &url.URL{Host: "[2001:db8:a0b:12f0::1]"}},
		{in: "[2001:db8:a0b:12f0::80]:80", out: &url.URL{Host: "[2001:db8:a0b:12f0::80]:80"}},
		{in: "[2001:db8:a0b:12f0::1]:8080", out: &url.URL{Host: "[2001:db8:a0b:12f0::1]:8080"}},
		{in: "example.com", out: &url.URL{Host: "example.com"}},
		{in: "1.example.com", out: &url.URL{Host: "1.example.com"}},
		{in: "subsub.sub.example.com", out: &url.URL{Host: "subsub.sub.example.com"}},
		{in: "subdomain_test.example.com", out: &url.URL{Host: "subdomain_test.example.com"}},
		{in: "user@example.com", out: &url.URL{Host: "example.com"}},
		{in: "user:passwd@example.com", out: &url.URL{Host: "example.com"}},
		{in: "https://user:passwd@subsub.sub.example.com", out: &url.URL{Scheme: "https", Host: "subsub.sub.example.com"}},
		{in: "http://user@example.com", out: &url.URL{Scheme: "http", Host: "example.com"}},
		{in: "hTTp://subSUB.sub.EXAMPLE.COM/x//////y///foo.mp3?c=z&a=x&b=y#t=20", out: &url.URL{Scheme: "http", Host: "subsub.sub.example.com", Path: "/x//////y///foo.mp3", RawQuery: "c=z&a=x&b=y", Fragment: "t=20"}},

		// Internationalized domain names.
		{in: "http://www.žluťoučký-kůň.cz/úpěl-ďábelské-ódy", err: false},
		{in: "http://www.xn--luouk-k-z2a6lsyxjlexh.cz/úpěl-ďábelské-ódy", err: false},
		{in: "http://żółć.pl/żółć.html", err: false},
		{in: "http://xn--kda4b0koi.pl/żółć.html", err: false},
		{in: "https://pressly.餐厅", err: false},
		{in: "https://pressly.组织机构", err: false},

		// Placeholder hosts parsed directly (no masking).
		{in: "http://{client}.local.com", out: &url.URL{Scheme: "http", Host: "{client}.local.com"}},
		{in: "{tenant}.local.com", out: &url.URL{Host: "{tenant}.local.com"}},
		{in: "http://{region}.{tenant}.host.com", out: &url.URL{Scheme: "http", Host: "{region}.{tenant}.host.com"}},
		{in: "{tenant}.local.com:8080", out: &url.URL{Host: "{tenant}.local.com:8080"}},

		// Malformed / unsupported inputs.
		{in: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAUAAAAFCAYAAACNbyblAAAAHElEQVQI12P4//8/w38GIAXDIBKE0DHxgljNBAAO9TXL0Y4OHwAAAABJRU5ErkJggg==", err: true},
		{in: "javascript:evilFunction()", err: true},
		{in: "otherscheme:garbage", err: true},
		{in: "<funnnytag>", err: true},
		{in: "http:/www.google.com", err: true},
		{in: "http:///www.google.com", err: true},
		{in: "javascript:void(0)", err: true},
		{in: "<script>", err: true},

		{in: hosts.Google.HTTP(), out: &url.URL{Scheme: "http", Host: "www.google.com"}},
		{in: "HTTP://WWW.GOOGLE.COM", out: &url.URL{Scheme: "http", Host: "www.google.com"}},
		{in: "HTTPS://WWW.google.COM", out: &url.URL{Scheme: "https", Host: "www.google.com"}},
	}

	for _, testCase := range tests {
		t.Run(testCase.in, func(t *testing.T) {
			parsed, err := urlt.Parse(testCase.in)

			if testCase.err {
				require.Error(t, err)
				assert.Nil(t, parsed)

				return
			}

			require.NoError(t, err)

			if testCase.out != nil {
				assert.Equal(t, testCase.out.Scheme, parsed.Scheme)
				assert.Equal(t, testCase.out.Host, parsed.Host)

				if testCase.out.Path != "" {
					assert.Equal(t, testCase.out.Path, parsed.Path)
				}

				if testCase.out.RawQuery != "" {
					assert.Equal(t, testCase.out.RawQuery, parsed.RawQuery)
				}

				if testCase.out.Fragment != "" {
					assert.Equal(t, testCase.out.Fragment, parsed.Fragment)
				}
			}

			// When no scheme is supplied, a default one must be injectable.
			if testCase.out != nil && !strings.Contains(strings.ToLower(testCase.in), "://") {
				withScheme, err := urlt.ParseWithDefaultScheme(testCase.in, "https")
				require.NoError(t, err)
				assert.Equal(t, "https", withScheme.Scheme)
			}
		})
	}
}

func TestParseWithDefaultScheme(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		scheme   string
		expected string
	}{
		{name: "injects scheme for bare host", in: "demo.com", scheme: "https", expected: "https"},
		{name: "injects scheme with port", in: "localhost:8080", scheme: "https", expected: "https"},
		{name: "injects scheme for scheme-relative url", in: "//demo.com", scheme: "http", expected: "http"},
		{name: "keeps existing scheme", in: "http://demo.com", scheme: "https", expected: "http"},
		{name: "default scheme for placeholder host", in: "{client}.demo.com", scheme: "https", expected: "https"},
		{name: "empty default keeps url scheme-less", in: "demo.com", scheme: "", expected: ""},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			parsed, err := urlt.ParseWithDefaultScheme(testCase.in, testCase.scheme)
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, parsed.Scheme)
		})
	}
}

func TestToString(t *testing.T) {
	t.Run("preserves placeholder braces", func(t *testing.T) {
		tests := map[string]string{
			"http://{client}.demo.com":                      "http://{client}.demo.com",
			"https://{tenant}.{region}.example.com/p?a=1#f": "https://{tenant}.{region}.example.com/p?a=1#f",
			"http://{tenant}.local.com:8080/p":              "http://{tenant}.local.com:8080/p",
			"{tenant}.local.com":                            "//{tenant}.local.com",
		}

		for in, expected := range tests {
			t.Run(in, func(t *testing.T) {
				parsed, err := urlt.Parse(in)
				require.NoError(t, err)
				assert.Equal(t, expected, urlt.ToString(parsed))
			})
		}
	})

	t.Run("matches stdlib for plain urls", func(t *testing.T) {
		tests := []string{
			"http://example.com",
			"https://example.com/path?x=1",
			"https://user:passwd@example.com/p#frag",
			"http://[2001:db8::1]:8080/p",
		}

		for _, in := range tests {
			t.Run(in, func(t *testing.T) {
				parsed, err := urlt.Parse(in)
				require.NoError(t, err)
				assert.Equal(t, parsed.String(), urlt.ToString(parsed))
			})
		}
	})

	t.Run("round-trips with Parse", func(t *testing.T) {
		tests := []string{
			"http://{client}.demo.com",
			"https://{tenant}.{region}.example.com/path?x=1",
			"http://{tenant}.local.com:8080/p",
			"https://example.com/path?x=1",
		}

		for _, in := range tests {
			t.Run(in, func(t *testing.T) {
				parsed, err := urlt.Parse(in)
				require.NoError(t, err)

				reparsed, err := urlt.Parse(urlt.ToString(parsed))
				require.NoError(t, err)

				assert.Equal(t, parsed.Host, reparsed.Host)
				assert.Equal(t, urlt.ToString(parsed), urlt.ToString(reparsed))
			})
		}
	})
}

func TestSplitHostPort(t *testing.T) {
	tests := []struct {
		host string
		want [2]string
		err  bool
	}{
		{host: "example.com", want: [2]string{"example.com", ""}},
		{host: "example.com:8080", want: [2]string{"example.com", "8080"}},
		{host: "[2001:db8::1]", want: [2]string{"[2001:db8::1]", ""}},
		{host: "[2001:db8::1]:80", want: [2]string{"[2001:db8::1]", "80"}},
		{host: "{tenant}.local.com:8080", want: [2]string{"{tenant}.local.com", "8080"}},
		{host: "example.com:", err: true},
		{host: "example.com:abc", err: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.host, func(t *testing.T) {
			host, port, err := urlt.SplitHostPort(&url.URL{Host: testCase.host})

			if testCase.err {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.want[0], host)
			assert.Equal(t, testCase.want[1], port)
		})
	}

	t.Run("nil url", func(t *testing.T) {
		_, _, err := urlt.SplitHostPort(nil)
		require.ErrorIs(t, err, urlt.ErrEmptyURL)
	})
}
