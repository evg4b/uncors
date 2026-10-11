package cli

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/stretchr/testify/assert"
)

func TestPrintRequests(t *testing.T) {
	requestURL, _ := url.Parse("http://localhost/api")

	bus := events.NewBus()
	subscription := bus.Subscribe(printerBuffer)

	bus.Publish(events.RequestStarted{ID: 1, Method: http.MethodGet, URL: requestURL, Handler: events.HandlerCache})
	bus.Publish(events.RequestCompleted{
		ID:      1,
		Handler: events.HandlerCache,
		Data:    &contracts.RequestData{Method: http.MethodGet, URL: requestURL, Code: http.StatusOK},
	})
	bus.Close()

	var output strings.Builder

	printRequests(subscription, tui.NewCliOutput(&output))

	lines := strings.Split(strings.TrimSpace(ansi.Strip(output.String())), "\n")
	assert.Len(t, lines, 1, "only completed requests are printed")
	assert.Contains(t, lines[0], "CACHE")
	assert.Contains(t, lines[0], "200 GET")
	assert.Contains(t, lines[0], "http://localhost/api")
}
