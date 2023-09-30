package urlreplacer_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

type replacerTestCase struct {
	name     string
	source   string
	expected string
}

func TestReplacerV2Replace(t *testing.T) {
	t.Run("url is not empty", func(t *testing.T) {
		t.Run("source", func(t *testing.T) {
			_, err := urlreplacer.NewReplacer("", hosts.Github.HTTP())

			assert.ErrorIs(t, err, urlreplacer.ErrEmptySourceURL)
		})

		t.Run("target", func(t *testing.T) {
			_, err := urlreplacer.NewReplacer(hosts.Localhost.Port(3000), "")

			assert.ErrorIs(t, err, urlreplacer.ErrEmptyTargetURL)
		})
	})

	t.Run("Replace", func(t *testing.T) {
		t.Run("where schemes given and", func(t *testing.T) {
			t.Run("schemes are equal", func(t *testing.T) {
				replacer, err := urlreplacer.NewReplacer("http://*.localhost.com", "http://api.*.com")
				testutils.CheckNoError(t, err)

				testsCases := []replacerTestCase{
					{
						name:     "url with scheme",
						source:   "http://test.localhost.com",
						expected: "http://api.test.com",
					},
					{
						name:     "url with path",
						source:   "http://test.localhost.com/api/config",
						expected: "http://api.test.com/api/config",
					},
					{
						name:     "url with path and query params",
						source:   "http://test.localhost.com/api/config?data=lorem",
						expected: "http://api.test.com/api/config?data=lorem",
					},
					{
						name:     "host only",
						source:   "test.localhost.com",
						expected: "api.test.com",
					},
				}
				for _, testsCase := range testsCases {
					t.Run(testsCase.name, func(t *testing.T) {
						actual, err := replacer.Replace(testsCase.source)

						assert.NoError(t, err)
						assert.Equal(t, testsCase.expected, actual)
					})
				}
			})

			t.Run("mapped from http to https", func(t *testing.T) {
				replacer, err := urlreplacer.NewReplacer("http://*.localhost.com", "https://api.*.com")
				testutils.CheckNoError(t, err)

				testsCases := []replacerTestCase{
					{
						name:     "url with scheme",
						source:   "http://test.localhost.com",
						expected: "https://api.test.com",
					},
					{
						name:     "url with path",
						source:   "http://test.localhost.com/api/config",
						expected: "https://api.test.com/api/config",
					},
					{
						name:     "url with path and query params",
						source:   "http://test.localhost.com/api/config?data=lorem",
						expected: "https://api.test.com/api/config?data=lorem",
					},
					{
						name:     "host only",
						source:   "test.localhost.com",
						expected: "api.test.com",
					},
				}
				for _, testsCase := range testsCases {
					t.Run(testsCase.name, func(t *testing.T) {
						actual, err := replacer.Replace(testsCase.source)

						assert.NoError(t, err)
						assert.Equal(t, testsCase.expected, actual)
					})
				}
			})

			t.Run("mapped from https to http", func(t *testing.T) {
				replacer, err := urlreplacer.NewReplacer("https://*.localhost.com", "http://api.*.com")
				testutils.CheckNoError(t, err)

				testsCases := []replacerTestCase{
					{
						name:     "url with scheme",
						source:   "https://test.localhost.com",
						expected: "http://api.test.com",
					},
					{
						name:     "url with path",
						source:   "https://test.localhost.com/api/config",
						expected: "http://api.test.com/api/config",
					},
					{
						name:     "url with path and query params",
						source:   "https://test.localhost.com/api/config?data=lorem",
						expected: "http://api.test.com/api/config?data=lorem",
					},
					{
						name:     "host only",
						source:   "test.localhost.com",
						expected: "api.test.com",
					},
				}
				for _, testsCase := range testsCases {
					t.Run(testsCase.name, func(t *testing.T) {
						actual, err := replacer.Replace(testsCase.source)

						assert.NoError(t, err)
						assert.Equal(t, testsCase.expected, actual)
					})
				}
			})
		})

		t.Run("where schemes are not given", func(t *testing.T) {
			testsCases := []replacerTestCase{
				{
					name:     "http url",
					source:   "http://test.localhost.com",
					expected: "http://api.test.com",
				},
				{
					name:     "https url",
					source:   "https://test.localhost.com",
					expected: "https://api.test.com",
				},
				{
					name:     "http url with path",
					source:   "http://test.localhost.com/api/config",
					expected: "http://api.test.com/api/config",
				},
				{
					name:     "https url with path",
					source:   "https://test.localhost.com/api/config",
					expected: "https://api.test.com/api/config",
				},
				{
					name:     "http url with path and query params",
					source:   "http://test.localhost.com/api/config?data=lorem",
					expected: "http://api.test.com/api/config?data=lorem",
				},
				{
					name:     "https url with path and query params",
					source:   "https://test.localhost.com/api/config?data=lorem",
					expected: "https://api.test.com/api/config?data=lorem",
				},
				{
					name:     "host only",
					source:   "test.localhost.com",
					expected: "api.test.com",
				},
			}

			t.Run("where schemes are not given", func(t *testing.T) {
				replacer, err := urlreplacer.NewReplacer("*.localhost.com", "api.*.com")
				testutils.CheckNoError(t, err)

				for _, testsCase := range testsCases {
					t.Run(testsCase.name, func(t *testing.T) {
						actual, err := replacer.Replace(testsCase.source)

						assert.NoError(t, err)
						assert.Equal(t, testsCase.expected, actual)
					})
				}
			})

			t.Run("where schemes set as //", func(t *testing.T) {
				replacer, err := urlreplacer.NewReplacer("//*.localhost.com", "//api.*.com")
				testutils.CheckNoError(t, err)
				for _, testsCase := range testsCases {
					t.Run(testsCase.name, func(t *testing.T) {
						actual, err := replacer.Replace(testsCase.source)

						assert.NoError(t, err)
						assert.Equal(t, testsCase.expected, actual)
					})
				}
			})
		})
	})
}

var isSecureTestCases = []struct {
	name     string
	url      string
	expected bool
}{
	{
		name:     "url with http scheme",
		url:      hosts.Localhost.HTTP(),
		expected: false,
	},
	{
		name:     "url with multiple scheme",
		url:      "//localhost",
		expected: false,
	},
	{
		name:     "url without scheme",
		url:      hosts.Localhost.Host(),
		expected: false,
	},
	{
		name:     "url with https scheme",
		url:      hosts.Localhost.HTTPS(),
		expected: true,
	},
}

func TestReplacerIsSourceSecure(t *testing.T) {
	makeReplacer := func(source string) *urlreplacer.Replacer {
		t.Helper()
		replacer, err := urlreplacer.NewReplacer(source, hosts.Github.HTTPS())
		if err != nil {
			t.Error(err)
		}

		return replacer
	}

	for _, testCase := range isSecureTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := makeReplacer(testCase.url).IsSourceSecure()

			assert.Equal(t, testCase.expected, actual)
		})
	}
}

func TestReplacerIsTargetSecure(t *testing.T) {
	makeReplacer := func(target string) *urlreplacer.Replacer {
		t.Helper()
		replacer, err := urlreplacer.NewReplacer(hosts.Github.HTTPS(), target)
		if err != nil {
			t.Error(err)
		}

		return replacer
	}

	for _, testCase := range isSecureTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := makeReplacer(testCase.url).IsTargetSecure()

			assert.Equal(t, testCase.expected, actual)
		})
	}
}

func TestReplacerIsMatched(t *testing.T) {
	replacer, err := urlreplacer.NewReplacer("*.my.cc:3000", "https://*.master-staging.com")
	testutils.CheckNoError(t, err)

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
			url:      "https//test.my.cc:3000",
			expected: true,
		},
		{
			name:     "matched domain with http scheme",
			url:      "http//test.my.cc:3000",
			expected: true,
		},
		{
			name:     "not matched to different domain",
			url:      "http//localhost",
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
