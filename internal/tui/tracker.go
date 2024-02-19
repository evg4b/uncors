package tui

import (
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
	URL    string
	Method string
}

type DoneRequestDefinition struct {
	RequestDefinition
	Status int
}

type RequestTracker struct {
	done     chan DoneRequestDefinition
	progress chan RequestDefinition
	requests map[string]RequestDefinition
	mutex    *sync.Mutex
}

func NewRequestTracker() RequestTracker {
	return RequestTracker{
		done:     make(chan DoneRequestDefinition, bufferSize),
		progress: make(chan RequestDefinition, bufferSize),
		requests: make(map[string]RequestDefinition),
		mutex:    &sync.Mutex{},
	}
}

func (r RequestTracker) Wrap(next contracts.Handler, prefix string) contracts.Handler {
	return contracts.HandlerFunc(func(writer contracts.ResponseWriter, request *contracts.Request) {
		responseWriter := NewResponseWriter(writer)
		uuid := r.registerRequest(request, prefix)
		defer func() {
			r.resolveRequest(uuid, responseWriter.StatusCode())
		}()
		next.ServeHTTP(responseWriter, request)
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
	def := RequestDefinition{
		Type:   prefix,
		URL:    request.URL.String(),
		Method: request.Method,
	}
	r.requests[uuid] = def
	r.progress <- def

	return uuid
}

func (r RequestTracker) resolveRequest(id string, w int) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.done <- DoneRequestDefinition{RequestDefinition: r.requests[id], Status: w}
	delete(r.requests, id)
}

func (r RequestTracker) Tick() tea.Msg {
	request := <-r.done

	return request
}

func (r RequestTracker) Tick2() tea.Msg {
	request := <-r.progress

	return request
}

func (r RequestTracker) View(spinner string) string {
	r.mutex.Lock()

	data := make([]string, 0, len(r.requests))
	for _, definition := range r.requests {
		builder := strings.Builder{}
		builder.WriteString(definition.Type)
		builder.WriteString(RenderRequest(definition, spinner))
		data = append(data, builder.String())
	}
	r.mutex.Unlock()
	sort.Strings(data)

	return strings.Join(data, "\n")
}

type ResponseWriter struct {
	writer contracts.ResponseWriter
}

func NewResponseWriter(writer contracts.ResponseWriter) ResponseWriter {
	return ResponseWriter{writer: writer}
}

func (r ResponseWriter) Header() http.Header {
	return r.writer.Header()
}

func (r ResponseWriter) Write(bytes []byte) (int, error) {
	return r.writer.Write(bytes)
}

func (r ResponseWriter) WriteHeader(statusCode int) {
	r.writer.WriteHeader(statusCode)
}

func (r ResponseWriter) StatusCode() int {
	return r.writer.StatusCode()
}
