package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/evg4b/uncors/inernal/handler"
)

var (
	target   = "github.com"
	protocol = "https"
	origin   = "localhost:4200"
)

func main() {
	flag.StringVar(&target, "target", target, "host:port to proxy requests to")
	flag.StringVar(&protocol, "protocol", protocol, "protocol used by the target")
	flag.StringVar(&origin, "origin", origin, "origin header to be used for the proxy request")

	flag.Parse()

	reqHandler := handler.NewRequestHandler(
		handler.WithOrigin(origin),
		handler.WithProtocol(protocol),
		handler.WithTarget(target),
	)

	http.HandleFunc("/", reqHandler.HandleRequest)

	log.Println("localhost:3000", "=>", target)
	http.ListenAndServe(":3000", nil)
}
