package request_tracker

import (
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/exp/maps"

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
	requests map[uuid.UUID]RequestDefinition
	mutex    *sync.Mutex
}

type ActiveRequests []RequestDefinition

func NewRequestTracker() RequestTracker {
	return RequestTracker{
		eventBus: make(chan tea.Msg, bufferSize),
		requests: make(map[uuid.UUID]RequestDefinition),
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

func (r RequestTracker) RegisterRequest(request *http.Request, prefix string) uuid.UUID {
	return r.registerRequest(request, prefix)
}

func (r RequestTracker) ResolveRequest(id uuid.UUID, statusCode int) {
	r.resolveRequest(id, statusCode)
}

func (r RequestTracker) registerRequest(request *http.Request, prefix string) uuid.UUID {
	id := helpers.GetUUID()
	r.mutex.Lock()
	defer r.mutex.Unlock()
	def := r.requestDefinition(request, prefix)
	r.requests[id] = def
	r.eventBus <- ActiveRequests(maps.Values(r.requests))

	return id
}

func (r RequestTracker) requestDefinition(request *http.Request, prefix string) RequestDefinition {
	host := request.URL.Scheme + "://" + request.URL.Host
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

func (r RequestTracker) resolveRequest(id uuid.UUID, status int) {
	r.eventBus <- DoneRequestDefinition{
		RequestDefinition: r.remove(id),
		Status:            status,
	}
}

func (r RequestTracker) remove(id uuid.UUID) RequestDefinition {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	resolved := r.requests[id]
	delete(r.requests, id)
	return resolved
}

func (r RequestTracker) Tick() tea.Msg {
	return <-r.eventBus
}

func (r RequestTracker) View(requests ActiveRequests, spinner string) string {
	data := make([]string, 0, len(requests))
	for _, definition := range requests {
		data = append(data, RenderRequest(definition, spinner))
	}
	sort.Strings(data)

	return strings.Join(data, "\n")
}

func (r RequestTracker) WrapHTTPClient(client contracts.HTTPClient, prefix string) contracts.HTTPClient {
	return HttpRequestTracker{
		tracker: r,
		client:  client,
		prefix:  prefix,
	}
}
