package version_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/internal/version"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

var errSome = errors.New("some error")

func TestCheckNewVersion(t *testing.T) {
	t.Run("do not panic where", func(t *testing.T) {
		tests := []struct {
			name    string
			client  contracts.HTTPClient
			version string
		}{
			{
				name: "http error is occupied",
				client: mocks.NewHTTPClientMock(t).
					DoMock.Return(nil, errSome),
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
			t.Run(testCase.name, func(t *testing.T) {
				assert.NotPanics(t, func() {
					output := &bytes.Buffer{}

					versionChecker := version.NewVersionChecker(
						version.WithOutput(tui.NewCliOutput(output)),
						version.WithHTTPClient(testCase.client),
						version.WithCurrentVersion(testCase.version),
					)

					versionChecker.CheckNewVersion(t.Context())

					outputData, err := io.ReadAll(output)
					testutils.CheckNoError(t, err)

					testutils.MatchSnapshot(t, string(outputData))
				})
			})
		}
	})

	t.Run("should panic when current version is incorrect", func(t *testing.T) {
		assert.Panics(t, func() {
			version.NewVersionChecker(
				version.WithCurrentVersion("#"),
			)
		})
	})

	versionResponse := func(version string) io.ReadCloser {
		return io.NopCloser(strings.NewReader(fmt.Sprintf(`{ "tag_name": "%s" }`, version)))
	}

	t.Run("should print info about new version", func(t *testing.T) {
		output := &bytes.Buffer{}

		httpClient := mocks.NewHTTPClientMock(t).
			DoMock.Return(&http.Response{Body: versionResponse("0.0.7")}, nil)

		versionChecker := version.NewVersionChecker(
			version.WithOutput(tui.NewCliOutput(output)),
			version.WithHTTPClient(httpClient),
			version.WithCurrentVersion("0.0.4"),
		)
		versionChecker.CheckNewVersion(t.Context())

		outputData, err := io.ReadAll(output)
		testutils.CheckNoError(t, err)

		testutils.MatchSnapshot(t, string(outputData))
	})

	t.Run("should not print info about same version", func(t *testing.T) {
		output := &bytes.Buffer{}

		httpClient := mocks.NewHTTPClientMock(t).
			DoMock.Return(&http.Response{Body: versionResponse("0.0.7")}, nil)

		versionChecker := version.NewVersionChecker(
			version.WithOutput(tui.NewCliOutput(output)),
			version.WithHTTPClient(httpClient),
			version.WithCurrentVersion("0.0.7"),
		)
		versionChecker.CheckNewVersion(t.Context())

		outputData, err := io.ReadAll(output)
		testutils.CheckNoError(t, err)

		testutils.MatchSnapshot(t, string(outputData))
	})

	t.Run("should print version check stub message", func(t *testing.T) {
		output := &bytes.Buffer{}

		versionChecker := version.NewVersionChecker(
			version.WithOutput(tui.NewCliOutput(output)),
			version.WithCurrentVersion("X.X.X"),
		)
		versionChecker.CheckNewVersion(t.Context())

		outputData, err := io.ReadAll(output)
		testutils.CheckNoError(t, err)

		testutils.MatchSnapshot(t, string(outputData))
	})
}
