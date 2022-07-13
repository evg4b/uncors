package proxy

import (
	"net/http"
	"strings"
)

type modificationsMap = map[string]func(string) (string, error)

func noop(s string) (string, error) { return s, nil }

func copyHeaders(from, to http.Header, modifications modificationsMap) error {
	for headerKey, headerValues := range from {
		transformedHeaderKey := strings.ToLower(headerKey)
		if transformedHeaderKey != "cookie" && transformedHeaderKey != "set-cookie" {
			modificationFunc, ok := modifications[transformedHeaderKey]
			if !ok {
				modificationFunc = noop
			}

			for _, value := range headerValues {
				updatedValue, err := modificationFunc(value)
				if err != nil {
					return err
				}

				to.Add(headerKey, updatedValue)
			}
		}
	}

	return nil
}
