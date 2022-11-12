package proxy

import (
	"net/http"
	"strings"
)

func (handler *Handler) makeOptionsResponse(writer http.ResponseWriter, req *http.Request) error {
	header := writer.Header()
	for key, values := range req.Header {
		lowerKey := strings.ToLower(key)
		if strings.Contains(lowerKey, "access-control-request") {
			for _, value := range values {
				transformedKey := strings.Replace(lowerKey, "request", "allow", 1)
				header.Add(transformedKey, value)
			}
		}
	}

	handler.logger.PrintResponse(&http.Response{
		StatusCode: http.StatusOK,
		Request:    req,
	})

	return nil
}
