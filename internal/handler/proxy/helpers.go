package proxy

import (
	"net/http"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/go-http-utils/headers"
)

var excluded = mapset.NewSet[string](
	headers.Cookie,
	headers.SetCookie,
	headers.ContentLength,
)

type modificationsMap = map[string]func(string) (string, error)

func noop(s string) (string, error) { return s, nil }

func copyHeaders(source, dest http.Header, modifications modificationsMap) error {
	for key, values := range source {
		if excluded.Contains(key) {
			continue
		}

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

	return nil
}
