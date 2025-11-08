package appbuilder

import (
	"context"
	"net/url"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/spf13/afero"
)

type Builder struct {
	t     *testing.T
	fs    afero.Fs
	uri   *url.URL
	https bool
}

func NewAppBuilder(t *testing.T) *Builder {
	t.Helper()

	return &Builder{t: t}
}

func (a *Builder) WithFs(fs afero.Fs) *Builder {
	a.t.Helper()
	a.fs = fs

	return a
}

func (a *Builder) WithHTTPS() *Builder {
	a.t.Helper()
	a.https = true

	return a
}

func (a *Builder) URI() *url.URL {
	a.t.Helper()

	return a.uri
}

func (a *Builder) Start(_ context.Context, _ *config.UncorsConfig) *uncors.App {
	a.t.Helper()

	panic("deda")
}
