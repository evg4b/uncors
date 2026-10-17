package server

import (
	"net/http"
	"time"
)

func SetShutdownTimeout(server *Server, timeout time.Duration) {
	server.shutdownTimeout = timeout
}

func HandleRequest(server *Server, handler http.Handler, writer http.ResponseWriter, request *http.Request) {
	server.handleRequest(handler, nil, writer, request)
}

const MaxCapturedBody = maxCapturedBody
