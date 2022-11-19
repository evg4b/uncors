package proxy

import (
	"net/http"
	"strings"

	"github.com/go-http-utils/headers"
)

type modificationsMap = map[string]func(string) (string, error)

func noop(s string) (string, error) { return s, nil }

func copyHeaders(source, dest http.Header, modifications modificationsMap) error {
	for key, values := range source {
		if !strings.EqualFold(key, headers.Cookie) && !strings.EqualFold(key, headers.SetCookie) {
			modificationFunc, ok := modifications[headers.Normalize(key)]
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
