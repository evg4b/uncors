package version_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/version"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/stretchr/testify/assert"
)

var errSome = errors.New("some error")

func versionResponse(tag string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(fmt.Sprintf(`{ "tag_name": %q }`, tag)))
}

func TestCheckNewVersion(t *testing.T) {
	t.Run("reports no new version when", func(t *testing.T) {
		tests := []struct {
			name   string
			client contracts.HTTPClient
		}{
			{
				name:   "the request fails",
				client: mocks.NewHTTPClientMock(t).DoMock.Return(nil, errSome),
			},
			{
				name: "the response is not json",
				client: mocks.NewHTTPClientMock(t).DoMock.Return(&http.Response{
					Body: io.NopCloser(strings.NewReader(`{ "version"`)),
				}, nil),
			},
			{
				name: "the tag is not a version",
				client: mocks.NewHTTPClientMock(t).DoMock.Return(&http.Response{
					Body: io.NopCloser(strings.NewReader(`{ "tag_name": "#" }`)),
				}, nil),
			},
			{
				name: "the latest release is the running one",
				client: mocks.NewHTTPClientMock(t).
					DoMock.Return(&http.Response{Body: versionResponse("0.0.3")}, nil),
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				checker := version.NewVersionChecker(
					version.WithHTTPClient(testCase.client),
					version.WithCurrentVersion("0.0.3"),
				)

				latest, available := checker.CheckNewVersion(t.Context())

				assert.False(t, available)
				assert.Empty(t, latest)
			})
		}
	})

	t.Run("reports a newer release", func(t *testing.T) {
		checker := version.NewVersionChecker(
			version.WithHTTPClient(mocks.NewHTTPClientMock(t).
				DoMock.Return(&http.Response{Body: versionResponse("0.0.7")}, nil)),
			version.WithCurrentVersion("0.0.4"),
		)

		latest, available := checker.CheckNewVersion(t.Context())

		assert.True(t, available)
		assert.Equal(t, "0.0.7", latest)
		assert.Equal(t, "0.0.4", checker.Current())
	})

	t.Run("skips the check for a development build", func(t *testing.T) {
		checker := version.NewVersionChecker(version.WithCurrentVersion("X.X.X"))

		latest, available := checker.CheckNewVersion(t.Context())

		assert.False(t, available)
		assert.Empty(t, latest)
		assert.Empty(t, checker.Current())
	})

	t.Run("skips the check when the current version is incorrect", func(t *testing.T) {
		checker := version.NewVersionChecker(version.WithCurrentVersion("#"))

		latest, available := checker.CheckNewVersion(t.Context())

		assert.False(t, available)
		assert.Empty(t, latest)
	})
}
