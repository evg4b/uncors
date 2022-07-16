package proxy

import (
	"net/http"
	"strings"
)

type modificationsMap = map[string]func(string) (string, error)

func noop(s string) (string, error) { return s, nil }

func copyHeaders(from, to http.Header, modifications modificationsMap) error {
	for key, values := range from {
		transformedKey := strings.ToLower(key)
		if transformedKey != "cookie" && transformedKey != "set-cookie" {
			modificationFunc, ok := modifications[transformedKey]
			if !ok {
				modificationFunc = noop
			}

			for _, value := range values {
				modifiedValue, err := modificationFunc(value)
				if err != nil {
					return err
				}

				to.Add(key, modifiedValue)
			}
		}
	}

	return nil
}
