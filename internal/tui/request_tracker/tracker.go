package request_tracker

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
)

const bufferSize = 10

type RequestDefinition struct {
	Type   string
	Host   string
	Path   string
	Params string
	Method string
}

type DoneRequestDefinition struct {
	RequestDefinition
	Status int
}

type RequestTracker struct {
	eventBus chan tea.Msg
	requests map[string]RequestDefinition
	mutex    *sync.Mutex
}

func NewRequestTracker() RequestTracker {
	return RequestTracker{
		eventBus: make(chan tea.Msg, bufferSize),
		requests: make(map[string]RequestDefinition),
		mutex:    &sync.Mutex{},
	}
}

func (r RequestTracker) Wrap(next contracts.Handler, prefix string) contracts.Handler {
	return contracts.HandlerFunc(func(writer contracts.ResponseWriter, request *contracts.Request) {
		uuid := r.registerRequest(request, prefix)
		defer func() {
			r.resolveRequest(uuid, writer.StatusCode())
		}()
		next.ServeHTTP(writer, request)
	})
}

func (r RequestTracker) RegisterRequest(request *http.Request, prefix string) string {
	return r.registerRequest(request, prefix)
}

func (r RequestTracker) ResolveRequest(id string, statusCode int) {
	r.resolveRequest(id, statusCode)
}

func (r RequestTracker) registerRequest(request *http.Request, prefix string) string {
	uuid := helpers.GetUUID()
	r.mutex.Lock()
	defer r.mutex.Unlock()
	def := r.funcName(request, prefix)
	r.requests[uuid] = def
	r.eventBus <- def

	return uuid
}

func (r RequestTracker) funcName(request *http.Request, prefix string) RequestDefinition {
	host := fmt.Sprintf("%s://%s", request.URL.Scheme, request.URL.Host)
	params := ""
	if request.URL.RawQuery != "" {
		params = "?" + request.URL.RawQuery
	}

	return RequestDefinition{
		Type:   prefix,
		Host:   host,
		Path:   request.URL.Path,
		Params: params,
		Method: request.Method,
	}
}

func (r RequestTracker) resolveRequest(id string, w int) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.eventBus <- DoneRequestDefinition{RequestDefinition: r.requests[id], Status: w}
	delete(r.requests, id)
}

func (r RequestTracker) Tick() tea.Msg {
	return <-r.eventBus
}

func (r RequestTracker) View(spinner string) string {
	r.mutex.Lock()

	data := make([]string, 0, len(r.requests))
	for _, definition := range r.requests {
		data = append(data, RenderRequest(definition, spinner))
	}
	r.mutex.Unlock()
	sort.Strings(data)

	return strings.Join(data, "\n")
}

func (r RequestTracker) WrapHttpClient(client contracts.HTTPClient, prefix string) contracts.HTTPClient {
	return HttpRequestTracker{
		tracker: r,
		client:  client,
		prefix:  prefix,
	}
}
