package infrastructure

import "net/http"

type HandlerFunc = func(http.ResponseWriter, *http.Request) error
type baseHandlerFunc = func(http.ResponseWriter, *http.Request)
