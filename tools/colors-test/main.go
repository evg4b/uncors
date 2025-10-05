//nolint:forbidigo,mnd
package main

import (
	"net/http"
	"net/url"
	"os"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/muesli/termenv"
	"github.com/samber/lo"
)

var (
	header = lipgloss.NewStyle().
		Width(100).
		Border(lipgloss.RoundedBorder(), true)
	subHeader = lipgloss.NewStyle().
			Width(100).
			Padding(0, 3)
)

func main() {
	infra.ConfigureLogger()
	log.SetColorProfile(termenv.TrueColor)
	log.SetLevel(log.DebugLevel)

	_, err := os.Stdout.Write([]byte{0x1B, 0x5B, 0x33, 0x3B, 0x4A, 0x1B, 0x5B, 0x48, 0x1B, 0x5B, 0x32, 0x4A})
	if err != nil {
		panic(err)
	}

	println(header.Render("LOGO:"))
	println(tui.Logo("X.X.X"))

	println(header.Render("DEFAULT LOGGER:"))
	testLogger(log.Default())

	println(header.Render("PROXY LOGGER:"))
	testLogger(uncors.NewProxyLogger(log.Default()))

	println(header.Render("CACHE LOGGER:"))
	testLogger(uncors.NewCacheLogger(log.Default()))

	println(header.Render("MOCK LOGGER:"))
	testLogger(uncors.NewMockLogger(log.Default()))

	println(header.Render("STATIC LOGGER:"))
	testLogger(uncors.NewStaticLogger(log.Default()))

	methods := []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
	}
	statuses := []int{
		http.StatusContinue,
		http.StatusOK,
		http.StatusFound,
		http.StatusBadRequest,
		http.StatusInternalServerError,
	}

	println(header.Render("REQUESTS PRINTING:"))
	lo.ForEach(methods, func(method string, _ int) {
		println(subHeader.Render(method + ":"))
		lo.ForEach(statuses, func(status int, _ int) {
			tui.PrintResponse(uncors.NewProxyLogger(log.Default()), makeRequest(method), status)
		})
	})

	println(header.Render("WARNING BOX:"))
	tui.PrintWarningBox(os.Stdout, "Warning message\nWarning message\nWarning message")

	println(header.Render("INFO BOX:"))
	tui.PrintInfoBox(os.Stdout, "Info message\nInfo message\nInfo message")
}

func testLogger(logger *log.Logger) {
	logger.Debug("Test debug")
	logger.With("key", "value").Debug("Test debug with key")
	logger.Info("Test info")
	logger.With("key", "value").Info("Test info with key")
	logger.Warn("Test warn")
	logger.With("key", "value").Warn("Test warn with key")
	logger.Error("Test error")
	logger.With("key", "value").Error("Test error with key")
}

func makeRequest(method string) *http.Request {
	uri, err := url.Parse(gofakeit.URL())
	if err != nil {
		panic(err)
	}

	return &http.Request{
		Method: method,
		URL:    uri,
	}
}
