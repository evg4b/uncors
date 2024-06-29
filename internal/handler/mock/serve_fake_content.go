package mock

import (
	"encoding/json"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/go-http-utils/headers"
)

func (h *Handler) serveFakeContent(writer contracts.ResponseWriter) error {
	response := h.response
	header := writer.Header()

	if len(header.Get(headers.ContentType)) == 0 {
		header.Set(headers.ContentType, "application/json")
	}

	data, err := response.Fake.Compile()
	if err != nil {
		return err
	}

	writer.WriteHeader(normaliseCode(response.Code))
	err = json.NewEncoder(writer).Encode(data)
	if err != nil {
		return err
	}

	return nil
}
