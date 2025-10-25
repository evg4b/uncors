//go:build release

package version_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/version"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestCheckNewVersion(t *testing.T) {
	t.Run("do not panic where", func(t *testing.T) {
		tests := []struct {
			name    string
			client  contracts.HTTPClient
			version string
		}{
			{
				name:    "current version is not correct",
				client:  mocks.NewHTTPClientMock(t),
				version: "#",
			},
			{
				name: "http error is occupied",
				client: mocks.NewHTTPClientMock(t).
					DoMock.Return(nil, errors.New("some http error")),
				version: "0.0.3",
			},
			{
				name: "invalid json received",
				client: mocks.NewHTTPClientMock(t).
					DoMock.Return(&http.Response{
					Body: io.NopCloser(strings.NewReader(`{ "version"`)),
				}, nil),
				version: "0.0.3",
			},
			{
				name: "incorrect json from api received",
				client: mocks.NewHTTPClientMock(t).
					DoMock.Return(&http.Response{
					Body: io.NopCloser(strings.NewReader(`{ "tag_name": "#" }`)),
				}, nil),
				version: "0.0.3",
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
				assert.NotPanics(t, func() {
					version.CheckNewVersion(context.Background(), testCase.client, testCase.version)

					outputData, err := io.ReadAll(output)
					testutils.CheckNoError(t, err)

					testutils.MatchSnapshot(t, string(outputData))
				})
			}))
		}
	})

	t.Run("should print ", func(t *testing.T) {
		t.Run("prop1", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			httpClient := mocks.NewHTTPClientMock(t).
				DoMock.Return(&http.Response{Body: io.NopCloser(strings.NewReader(`{ "tag_name": "0.0.7" }`))}, nil)

			version.CheckNewVersion(context.Background(), httpClient, "0.0.4")

			outputData, err := io.ReadAll(output)
			testutils.CheckNoError(t, err)

			testutils.MatchSnapshot(t, string(outputData))
		}))

		t.Run("prop2", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			httpClient := mocks.NewHTTPClientMock(t).
				DoMock.Return(&http.Response{Body: io.NopCloser(strings.NewReader(`{ "tag_name": "0.0.7" }`))}, nil)

			version.CheckNewVersion(context.Background(), httpClient, "0.0.7")

			outputData, err := io.ReadAll(output)
			testutils.CheckNoError(t, err)

			testutils.MatchSnapshot(t, string(outputData))
		}))
	})
}
