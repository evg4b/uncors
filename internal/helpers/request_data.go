package helpers

import (
	"github.com/evg4b/uncors/internal/contracts"
)

func ToRequestData(req *contracts.Request, code int) *contracts.RequestData {
	return &contracts.RequestData{
		Method: req.Method,
		URL:    req.URL,
		Header: req.Header,
		Body:   nil,
		Code:   code,
	}
}
