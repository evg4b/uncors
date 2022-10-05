package proxy

import (
	"net/http"
	"strings"
)

type modificationsMap = map[string]func(string) (string, error)

func noop(s string) (string, error) { return s, nil }

func copyHeaders(source, dest http.Header, modifications modificationsMap) error {
	for key, values := range source {
		if !strings.EqualFold(key, "cookie") && !strings.EqualFold(key, "set-cookie") {
			modificationFunc, ok := modifications[strings.ToLower(key)]
			if !ok {
				modificationFunc = noop
			}

			for _, value := range values {
				modifiedValue, err := modificationFunc(value)
				if err != nil {
					return err
				}

				dest.Add(key, modifiedValue)
			}
		}
	}

	return nil
}
