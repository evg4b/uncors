// nolint: lll
package urlglob_test

import (
	"testing"

	"github.com/evg4b/uncors/pkg/urlglob"
	"github.com/stretchr/testify/assert"
)

func TestNewURLGlob(t *testing.T) {
	t.Run("return error when", func(t *testing.T) {
		tests := []struct {
			name     string
			rawURL   string
			errorMsg string
		}{
			{
				name:     "url is empty",
				rawURL:   "",
				errorMsg: "url should not be empty",
			},
			{
				name:     "url is invalid",
				rawURL:   "&*%",
				errorMsg: "failed to craete glob from '&*%': invalid url: parse \"&*%\": invalid URL escape \"%\"",
			},
			{
				name:     "sheme contains wildcard",
				rawURL:   "http*://demo.com",
				errorMsg: "failed to craete glob from 'http*://demo.com': invalid url: parse \"http*://demo.com\": first path segment in URL cannot contain colon",
			},
			{
				name:     "pattern contains path",
				rawURL:   "https://demo.com/api/info",
				errorMsg: "failed to craete glob from 'https://demo.com/api/info': url pattern should not contain path, query or fragment",
			},
			{
				name:     "pattern contains query",
				rawURL:   "https://demo.com?demo=data",
				errorMsg: "failed to craete glob from 'https://demo.com?demo=data': url pattern should not contain path, query or fragment",
			},
			{
				name:     "pattern contains fragment",
				rawURL:   "https://demo.com#target",
				errorMsg: "failed to craete glob from 'https://demo.com#target': url pattern should not contain path, query or fragment",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				actual, err := urlglob.NewURLGlob(tt.rawURL)

				assert.Nil(t, actual)
				assert.EqualError(t, err, tt.errorMsg)
			})
		}
	})
}

func TestURLGlobMatch(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		URL      string
		expected bool
	}{
		{
			name:     "matched url with scheme",
			pattern:  "https://demo.com",
			URL:      "https://demo.com/demo/test",
			expected: true,
		},
		{
			name:     "not matched url with incorrect scheme",
			pattern:  "https://demo.com",
			URL:      "http://demo.com/demo/test",
			expected: false,
		},
		{
			name:     "not matched url with incorrect scheme",
			pattern:  "http://demo.com",
			URL:      "https://demo.com/demo/test",
			expected: false,
		},
		{
			name:     "not matched http for pattern withut scheme",
			pattern:  "//demo.com",
			URL:      "http://demo.com/demo/test",
			expected: true,
		},
		{
			name:     "not matched https for pattern withut scheme",
			pattern:  "//demo.com",
			URL:      "https://demo.com/demo/test",
			expected: true,
		},
		{
			name:     "scheme mispatch",
			pattern:  "https://base.localhost.com",
			URL:      "http://base.localhost.com/api/info",
			expected: false,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			glob, err := urlglob.NewURLGlob(testCase.pattern)
			if err != nil {
				t.Fatal(err)
			}

			actual, err := glob.MatchString(testCase.URL)

			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, actual)
		})
	}
}

func TestURLGlobReplaceAllString(t *testing.T) {
	t.Run("base config", func(t *testing.T) {
		tests := []struct {
			name     string
			pattern  string
			URL      string
			repl     string
			expected string
		}{
			{
				name:     "correctly transform http to https",
				pattern:  "http://*.my.cc",
				URL:      "http://test.my.cc/404",
				repl:     "https://*.realapi.com",
				expected: "https://test.realapi.com/404",
			},
			{
				name:     "correctly transform https to http",
				pattern:  "https://*.my.cc",
				URL:      "https://test.my.cc/cc",
				repl:     "http://*.realapi.com",
				expected: "http://test.realapi.com/cc",
			},
			{
				name:     "correctly copy scheme from original url",
				pattern:  "https://*.my.cc",
				URL:      "https://test.my.cc/api/test",
				repl:     "//*.realapi.com",
				expected: "https://test.realapi.com/api/test",
			},
			{
				name:     "correctly replace when repl has no wildcard",
				pattern:  "https://*.my.cc",
				URL:      "https://test.my.cc/api/info",
				repl:     "https://static.com",
				expected: "https://static.com/api/info",
			},
			{
				name:     "correctly remove port",
				pattern:  "http://*.my.cc:3000",
				URL:      "http://test.my.cc:3000",
				repl:     "https://*.realapi.com",
				expected: "https://test.realapi.com",
			},
			{
				name:     "correctly add port",
				pattern:  "http://*.my.cc",
				URL:      "http://test.my.cc/test.html",
				repl:     "https://*.realapi.com:8080",
				expected: "https://test.realapi.com:8080/test.html",
			},
			{
				name:     "correctly change port",
				pattern:  "http://*.my.cc:7600",
				URL:      "http://test.my.cc:7600/test.html",
				repl:     "https://*.realapi.com:8080",
				expected: "https://test.realapi.com:8080/test.html",
			},
			{
				name:     "correctly handle dinamic port",
				pattern:  "http://*.my.cc",
				URL:      "http://test.my.cc:7600/test.html",
				repl:     "https://*.realapi.com",
				expected: "https://test.realapi.com/test.html",
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				glob, err := urlglob.NewURLGlob(testCase.pattern)
				if err != nil {
					t.Fatal(err)
				}

				repl, err := urlglob.NewReplacePatternString(testCase.repl)
				if err != nil {
					t.Fatal(err)
				}

				actual, err := glob.ReplaceAllString(testCase.URL, repl)

				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, actual)
			})
		}
	})
}
