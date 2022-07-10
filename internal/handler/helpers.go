package handler

import (
	"fmt"
	"net/http"
	"strings"
)

func getUrl(req *http.Request) string {
	val := req.Context().Value("protocol")
	protocol := val.(string)
	return fmt.Sprintf("%s://%s%s", protocol, req.Host, req.URL.String())
}

type replcaeFunc = func(string) (string, error)

func copyHeaders(from, to http.Header, replcaer replcaeFunc, headers []string) error {
	for key, values := range from {
		if strings.ToLower(key) == "cookie" {
			continue
		}

		for _, value := range values {
			if contains(headers, key) {
				var err error
				value, err = replcaer(value)
				if err != nil {
					return err
				}

			}

			to.Add(key, value)
		}
	}

	return nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
