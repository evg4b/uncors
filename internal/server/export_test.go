package server

import "time"

func SetShutdownTimeout(server *Server, timeout time.Duration) {
	server.shutdownTimeout = timeout
}
