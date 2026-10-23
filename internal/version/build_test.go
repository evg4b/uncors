package version_test

import (
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/evg4b/uncors/internal/version"
	"github.com/stretchr/testify/assert"
)

const revision = "0123456789abcdef0123456789abcdef01234567"

func buildInfo(settings ...debug.BuildSetting) *debug.BuildInfo {
	return &debug.BuildInfo{GoVersion: "go1.99.0", Settings: settings}
}

func TestNewBuild(t *testing.T) {
	tests := []struct {
		name    string
		release string
		info    *debug.BuildInfo
		version string
		commit  string
		goVer   string
	}{
		{
			name:    "release build",
			release: "v1.2.3",
			info:    buildInfo(debug.BuildSetting{Key: "vcs.revision", Value: revision}),
			version: "v1.2.3",
			commit:  revision,
			goVer:   "go1.99.0",
		},
		{
			name:    "clean build without a release version",
			info:    buildInfo(debug.BuildSetting{Key: "vcs.revision", Value: revision}),
			version: "0123456",
			commit:  revision,
			goVer:   "go1.99.0",
		},
		{
			name: "dirty build without a release version",
			info: buildInfo(
				debug.BuildSetting{Key: "vcs.revision", Value: revision},
				debug.BuildSetting{Key: "vcs.modified", Value: "true"},
			),
			version: "0123456-dirty",
			commit:  revision + "-dirty",
			goVer:   "go1.99.0",
		},
		{
			name:    "build without vcs information",
			info:    buildInfo(),
			version: "dev",
			commit:  "unknown",
			goVer:   "go1.99.0",
		},
		{
			name:    "no build information",
			version: "dev",
			commit:  "unknown",
			goVer:   runtime.Version(),
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			build := version.NewBuild(testCase.release, testCase.info)

			assert.Equal(t, testCase.version, build.Version())
			assert.Equal(t, testCase.commit, build.Commit())
			assert.Equal(t, testCase.goVer, build.GoVersion)
		})
	}
}

func TestBuildString(t *testing.T) {
	build := version.Build{Release: "v1.2.3", GoVersion: "go1.99.0", Revision: revision, Modified: true}

	assert.Equal(t, "version: v1.2.3\ngo:      go1.99.0\ncommit:  "+revision+"-dirty\n", build.String())
}
