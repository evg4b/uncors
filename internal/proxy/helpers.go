package proxy

import (
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/responseprinter"
	"github.com/pterm/pterm"
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

func makeOptionsResponse(printer pterm.PrefixPrinter, writer http.ResponseWriter, req *http.Request) error {
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

	printer.Printfln(responseprinter.PrintResponse(&http.Response{
		StatusCode: http.StatusOK,
		Request:    req,
	}))

	return nil
}
