package server

import (
	"net/http"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
)

func SetShutdownTimeout(server *Server, timeout time.Duration) {
	server.shutdownTimeout = timeout
}

func HandleRequest(server *Server, handler contracts.Handler, writer http.ResponseWriter, request *http.Request) {
	server.handleRequest(handler, writer, request)
}
