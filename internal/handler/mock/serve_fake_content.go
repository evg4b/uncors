package mock

import (
	"encoding/json"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/go-http-utils/headers"
)

func (h *Handler) serveFakeContent(writer contracts.ResponseWriter, _request *contracts.Request) {
	response := h.response
	header := writer.Header()

	if len(header.Get(headers.ContentType)) == 0 {
		header.Set(headers.ContentType, "application/json")
	}

	writer.WriteHeader(normaliseCode(response.Code))
	data, err := response.Fake.Compile()
	if err != nil {
		panic(err)
	}
	err = json.NewEncoder(writer).Encode(data)
	if err != nil {
		panic(err)
	}
}
