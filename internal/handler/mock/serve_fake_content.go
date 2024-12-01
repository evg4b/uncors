package mock

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/go-http-utils/headers"
)

var ErrInvalidSeed = errors.New("invalid $__uncors__seed parameter")

const seedKeyName = "$__uncors__seed"

func (h *Handler) serveFakeContent(writer contracts.ResponseWriter, request *contracts.Request) error {
	response := h.response
	header := writer.Header()

	if len(header.Get(headers.ContentType)) == 0 {
		header.Set(headers.ContentType, "application/json")
	}

	seed, err := extractSeed(response, request)
	if err != nil {
		return err
	}

	data, err := h.generator.Generate(response.Fake, seed)
	if err != nil {
		return err
	}

	writer.WriteHeader(normaliseCode(response.Code))

	return json.NewEncoder(writer).
		Encode(data)
}

func extractSeed(response config.Response, request *contracts.Request) (uint64, error) {
	if response.Seed > 0 {
		return response.Seed, nil
	}

	queries := request.URL.Query()
	if queries.Has(seedKeyName) {
		return parseUint(queries.Get(seedKeyName))
	}

	header := request.Header.Get(seedKeyName)
	if header != "" {
		return parseUint(header)
	}

	return 0, nil
}

func parseUint(value string) (uint64, error) {
	seed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, errors.Join(ErrInvalidSeed, err)
	}

	return seed, nil
}
