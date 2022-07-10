package handler

import (
	"fmt"
	"net/http"
)

func getUrl(req *http.Request) string {
	val := req.Context().Value("protocol")
	protocol := val.(string)
	return fmt.Sprintf("%s://%s%s", protocol, req.Host, req.URL.String())
}
