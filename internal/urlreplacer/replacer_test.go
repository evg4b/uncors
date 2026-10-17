package urlreplacer_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReplacerErrors(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		target   string
		expected error
	}{
		{name: "empty source", target: hosts.Github.HTTP().String(), expected: urlreplacer.ErrEmptySourceURL},
		{name: "empty target", source: hosts.Localhost.Port(3000).String(), expected: urlreplacer.ErrEmptyTargetURL},
		{
			name:     "duplicate source key",
			source:   "http://{client}.{client}.com",
			target:   "http://api.com",
			expected: urlreplacer.ErrDuplicateSourceKey,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := urlreplacer.NewReplacer(testCase.source, testCase.target)

			require.ErrorIs(t, err, testCase.expected)
		})
	}
}

func TestReplacerReplace(t *testing.T) {
	tests := []struct {
		name   string
		source string
		target string
		cases  map[string]string
	}{
		{
			name:   "http to http",
			source: "http://{tenant}.localhost.com",
			target: "http://api.{tenant}.com",
			cases: map[string]string{
				"http://test.localhost.com":                       "http://api.test.com",
				"http://test.localhost.com/api/config?data=lorem": "http://api.test.com/api/config?data=lorem",
				"test.localhost.com":                              "api.test.com",
			},
		},
		{
			name:   "http to https",
			source: "http://{tenant}.localhost.com",
			target: "https://api.{tenant}.com",
			cases: map[string]string{
				"http://test.localhost.com":                       "https://api.test.com",
				"http://test.localhost.com/api/config?data=lorem": "https://api.test.com/api/config?data=lorem",
				"test.localhost.com":                              "api.test.com",
			},
		},
		{
			name:   "https to http",
			source: "https://{tenant}.localhost.com",
			target: "http://api.{tenant}.com",
			cases: map[string]string{
				"https://test.localhost.com":                       "http://api.test.com",
				"https://test.localhost.com/api/config?data=lorem": "http://api.test.com/api/config?data=lorem",
				"test.localhost.com":                               "api.test.com",
			},
		},
		{
			name:   "without schemes",
			source: "{tenant}.localhost.com",
			target: "api.{tenant}.com",
			cases:  schemelessCases,
		},
		{
			name:   "with // schemes",
			source: "//{tenant}.localhost.com",
			target: "//api.{tenant}.com",
			cases:  schemelessCases,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			replacer, err := urlreplacer.NewReplacer(testCase.source, testCase.target)
			require.NoError(t, err)

			for source, expected := range testCase.cases {
				actual, err := replacer.Replace(source)

				require.NoError(t, err)
				assert.Equal(t, expected, actual, source)
			}
		})
	}
}

func TestReplacerPlaceholderMatchesOneLabel(t *testing.T) {
	replacer, err := urlreplacer.NewReplacer("http://{tenant}.local.dev:3000", "https://{tenant}.api.com")
	require.NoError(t, err)

	t.Run("replaces only the host", func(t *testing.T) {
		cases := map[string]string{
			"http://acme.local.dev:3000":                          "https://acme.api.com",
			"http://acme.local.dev:3000/api/v1?page=2":            "https://acme.api.com/api/v1?page=2",
			"http://acme.local.dev:3000/files/report.local.dev":   "https://acme.api.com/files/report.local.dev",
			"http://acme.local.dev:3000/?q=.local.dev":            "https://acme.api.com/?q=.local.dev",
			"http://acme.local.dev:3000/a.local.dev?b=.local.dev": "https://acme.api.com/a.local.dev?b=.local.dev",
			"acme.local.dev": "acme.api.com",
		}

		for source, expected := range cases {
			actual, err := replacer.Replace(source)

			require.NoError(t, err, source)
			assert.Equal(t, expected, actual, source)
		}
	})

	t.Run("does not match a placeholder across labels or into the url", func(t *testing.T) {
		urls := []string{
			"http://a.b.local.dev:3000",
			"http://other.host/x.local.dev",
			"http://other.host:3000?q=.local.dev",
			"http://user@other.host/.local.dev",
			"http://.local.dev:3000",
		}

		for _, url := range urls {
			assert.False(t, replacer.IsMatched(url), url)
		}
	})

	t.Run("keeps the leading dot of a cookie domain", func(t *testing.T) {
		reverse, err := urlreplacer.NewReplacer("https://{tenant}.api.com", "http://{tenant}.local.dev")
		require.NoError(t, err)

		assert.Equal(t, ".acme.local.dev", reverse.ReplaceSoft(".acme.api.com"))
		assert.Equal(t, "acme.local.dev", reverse.ReplaceSoft("acme.api.com"))
	})
}

var schemelessCases = map[string]string{
	"http://test.localhost.com":                        "http://api.test.com",
	"https://test.localhost.com":                       "https://api.test.com",
	"http://test.localhost.com/api/config?data=lorem":  "http://api.test.com/api/config?data=lorem",
	"https://test.localhost.com/api/config?data=lorem": "https://api.test.com/api/config?data=lorem",
	"test.localhost.com":                               "api.test.com",
}

var isSecureTestCases = []struct {
	name     string
	url      string
	expected bool
}{
	{
		name:     "url with http scheme",
		url:      hosts.Localhost.HTTP().String(),
		expected: false,
	},
	{
		name:     "url with multiple scheme",
		url:      "//localhost",
		expected: false,
	},
	{
		name:     "url without scheme",
		url:      hosts.Localhost.Host().String(),
		expected: false,
	},
	{
		name:     "url with https scheme",
		url:      hosts.Localhost.HTTPS().String(),
		expected: true,
	},
}

func TestReplacerIsTargetSecure(t *testing.T) {
	for _, testCase := range isSecureTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			replacer, err := urlreplacer.NewReplacer(hosts.Github.HTTPS().String(), testCase.url)
			require.NoError(t, err)

			assert.Equal(t, testCase.expected, replacer.IsTargetSecure())
		})
	}
}

func TestReplacerReplaceSoft(t *testing.T) {
	replacer, err := urlreplacer.NewReplacer("http://local.com", "https://api.com")
	require.NoError(t, err)

	t.Run("returns replaced url when source matches", func(t *testing.T) {
		result := replacer.ReplaceSoft("http://local.com/path")
		assert.Equal(t, "https://api.com/path", result)
	})

	t.Run("returns original url when source does not match", func(t *testing.T) {
		result := replacer.ReplaceSoft("http://other.com/path")
		assert.Equal(t, "http://other.com/path", result)
	})
}

func TestReplacerIsMatched(t *testing.T) {
	replacer, err := urlreplacer.NewReplacer("{tenant}.my.cc:3000", "https://{tenant}.master-staging.com")
	require.NoError(t, err)

	testsCases := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "domain without scheme",
			url:      "premium.my.cc:3000",
			expected: true,
		},
		{
			name:     "matched domain with different port",
			url:      "premium.my.cc:2900",
			expected: true,
		},
		{
			name:     "matched domain without port",
			url:      "standard.my.cc",
			expected: true,
		},
		{
			name:     "matched domain with same scheme and correct port",
			url:      "//test.my.cc:3000",
			expected: true,
		},
		{
			name:     "matched domain with https scheme",
			url:      "https://test.my.cc:3000",
			expected: true,
		},
		{
			name:     "matched domain with http scheme",
			url:      "http://test.my.cc:3000",
			expected: true,
		},
		{
			name:     "not matched to different domain",
			url:      "http://localhost",
			expected: false,
		},
	}
	for _, testsCase := range testsCases {
		t.Run(testsCase.name, func(t *testing.T) {
			actual := replacer.IsMatched(testsCase.url)

			assert.Equal(t, testsCase.expected, actual)
		})
	}
}
