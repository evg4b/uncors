package proxy

import (
	"net/http"
	"strings"
)

type modificationsMap = map[string]func(string) (string, error)

func noop(s string) (string, error) { return s, nil }

func copyHeaders(from, to http.Header, modifications modificationsMap) error {
	for headerKey, headerValues := range from {
		if strings.ToLower(headerKey) != "cookie" && strings.ToLower(headerKey) != "set-cookie" {
			modificationFunc, ok := modifications[headerKey]
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
