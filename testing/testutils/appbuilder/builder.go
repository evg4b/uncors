package appbuilder

import (
	"context"
	"io"
	"net/url"
	"testing"
	"time"

	"github.com/charmbracelet/log"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
)

const delay = 10 * time.Millisecond

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

func (a *Builder) Start(ctx context.Context, config *config.UncorsConfig) *uncors.App {
	a.t.Helper()
	app := uncors.CreateApp(a.fs, log.New(io.Discard), "x.x.x")
	go app.Start(ctx, config)
	time.Sleep(delay)
	var err error
	a.uri, err = url.Parse(a.prefix() + a.addr(app))
	testutils.CheckNoError(a.t, err)

	return app
}

func (a *Builder) addr(app *uncors.App) string {
	a.t.Helper()
	addr := app.HTTPAddr().String()
	if a.https {
		addr = app.HTTPSAddr().String()
	}

	return addr
}

func (a *Builder) prefix() string {
	a.t.Helper()
	if a.https {
		return "https://"
	}

	return "http://"
}
