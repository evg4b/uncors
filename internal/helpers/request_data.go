package helpers

import (
	"github.com/evg4b/uncors/internal/contracts"
)

func ToRequestData(req *contracts.Request, res contracts.ResponseWriter) *contracts.ReqestData {
	return &contracts.ReqestData{
		Method: req.Method,
		URL:    req.URL,
		Header: req.Header,
		Body:   nil,
		Code:   res.StatusCode(),
	}
}
