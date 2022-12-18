//go:build release

package ui_test

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/log"
	"github.com/evg4b/uncors/internal/ui"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestCheckLastVersion(t *testing.T) {
	log.DisableColor()
	log.EnableDebugMessages()

	t.Run("do not panic where", func(t *testing.T) {
		tests := []struct {
			name     string
			client   contracts.HTTPClient
			version  string
			expected string
		}{
			{
				name:     "currect version is not correct",
				client:   mocks.NewHttpClientMock(t),
				version:  "#",
				expected: "   DEBUG  Checking new version\n   DEBUG  failed to parse currect version: Malformed version: #\n",
			},
			{
				name: "http error is occuped",
				client: mocks.NewHttpClientMock(t).
					DoMock.Return(nil, errors.New("some http error")),
				version:  "0.0.3",
				expected: "   DEBUG  Checking new version\n   DEBUG  http error ocupted: some http error\n",
			},
			{
				name: "invalid json received",
				client: mocks.NewHttpClientMock(t).
					DoMock.Return(&http.Response{
					Body: io.NopCloser(strings.NewReader(`{ "version"`)),
				}, nil),
				version:  "0.0.3",
				expected: "   DEBUG  Checking new version\n   DEBUG  failed to parse last version respoce: unexpected EOF\n",
			},
			{
				name: "incorrect json from api received",
				client: mocks.NewHttpClientMock(t).
					DoMock.Return(&http.Response{
					Body: io.NopCloser(strings.NewReader(`{ "tag_name": "#" }`)),
				}, nil),
				version:  "0.0.3",
				expected: "   DEBUG  Checking new version\n   DEBUG  failed to parse last version: Malformed version: #\n",
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
				assert.NotPanics(t, func() {
					ui.CheckLastVersion(testCase.client, testCase.version)

					outputData, err := ioutil.ReadAll(output)
					testutils.CheckNoError(t, err)

					assert.Equal(t, testCase.expected, string(outputData))
				})
			}))
		}
	})

	t.Run("should print ", func(t *testing.T) {
		t.Run("prop1", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			httpClient := mocks.NewHttpClientMock(t).
				DoMock.Return(&http.Response{Body: io.NopCloser(strings.NewReader(`{ "tag_name": "0.0.7" }`))}, nil)

			ui.CheckLastVersion(httpClient, "0.0.4")

			outputData, err := ioutil.ReadAll(output)
			testutils.CheckNoError(t, err)
			expected := `   DEBUG  Checking new version
    INFO  NEW VERSION IS Available!
          0.0.4 is not latest, you should upgrade to 0.0.7.
          See more information on https://github.com/evg4b/uncors/releases

`
			assert.Equal(t, expected, string(outputData))
		}))

		t.Run("prop2", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			httpClient := mocks.NewHttpClientMock(t).
				DoMock.Return(&http.Response{Body: io.NopCloser(strings.NewReader(`{ "tag_name": "0.0.7" }`))}, nil)

			ui.CheckLastVersion(httpClient, "0.0.7")

			outputData, err := ioutil.ReadAll(output)
			testutils.CheckNoError(t, err)
			expected := "   DEBUG  Checking new version\n   DEBUG  Version is up to date\n"
			assert.Equal(t, expected, string(outputData))
		}))
	})
}