package urlreplacer_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/urlreplacer"
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
			_, err := urlreplacer.NewReplacerV2("", "http://github.com")

			assert.ErrorIs(t, err, urlreplacer.ErrEmptySourceURL)
		})

		t.Run("target", func(t *testing.T) {
			_, err := urlreplacer.NewReplacerV2("localhost:3000", "")

			assert.ErrorIs(t, err, urlreplacer.ErrEmptyTargetURL)
		})
	})

	t.Run("replace", func(t *testing.T) {
		t.Run("where schemes given and", func(t *testing.T) {
			t.Run("schemes are equal", func(t *testing.T) {
				replacer, err := urlreplacer.NewReplacerV2("http://*.localhost.com", "http://api.*.com")
				testutils.CheckNoError(t, err)

				testsCases := []replacerTestCase{
					{
						name:     "url with sheme",
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
				replacer, err := urlreplacer.NewReplacerV2("http://*.localhost.com", "https://api.*.com")
				testutils.CheckNoError(t, err)

				testsCases := []replacerTestCase{
					{
						name:     "url with sheme",
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
				replacer, err := urlreplacer.NewReplacerV2("https://*.localhost.com", "http://api.*.com")
				testutils.CheckNoError(t, err)

				testsCases := []replacerTestCase{
					{
						name:     "url with sheme",
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
				replacer, err := urlreplacer.NewReplacerV2("*.localhost.com", "api.*.com")
				testutils.CheckNoError(t, err)

				for _, testsCase := range testsCases {
					t.Run(testsCase.name, func(t *testing.T) {
						actual, err := replacer.Replace(testsCase.source)

						assert.NoError(t, err)
						assert.Equal(t, testsCase.expected, actual)
					})
				}
			})

			t.Run("where schemes setted as //", func(t *testing.T) {
				replacer, err := urlreplacer.NewReplacerV2("//*.localhost.com", "//api.*.com")
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
