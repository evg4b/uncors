package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

const (
	shortRevisionLength = 7
	devVersion          = "dev"
	unknownRevision     = "unknown"
	dirtySuffix         = "-dirty"
)

type Build struct {
	Release   string
	GoVersion string
	Revision  string
	Modified  bool
}

func ReadBuild(release string) Build {
	info, _ := debug.ReadBuildInfo()

	return NewBuild(release, info)
}

func NewBuild(release string, info *debug.BuildInfo) Build {
	build := Build{
		Release:   release,
		GoVersion: runtime.Version(),
	}

	if info == nil {
		return build
	}

	if info.GoVersion != "" {
		build.GoVersion = info.GoVersion
	}

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			build.Revision = setting.Value
		case "vcs.modified":
			build.Modified = setting.Value == "true"
		}
	}

	return build
}

func (b Build) Version() string {
	if b.Release != "" {
		return b.Release
	}

	if b.Revision == "" {
		return devVersion
	}

	revision := b.Revision
	if len(revision) > shortRevisionLength {
		revision = revision[:shortRevisionLength]
	}

	if b.Modified {
		revision += dirtySuffix
	}

	return revision
}

func (b Build) Commit() string {
	if b.Revision == "" {
		return unknownRevision
	}

	if b.Modified {
		return b.Revision + dirtySuffix
	}

	return b.Revision
}

func (b Build) String() string {
	var builder strings.Builder

	_, _ = fmt.Fprintf(&builder, "version: %s\n", b.Version())
	_, _ = fmt.Fprintf(&builder, "go:      %s\n", b.GoVersion)
	_, _ = fmt.Fprintf(&builder, "commit:  %s\n", b.Commit())

	return builder.String()
}
