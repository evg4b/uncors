package fakedata

import (
	"errors"
	"strconv"

	"github.com/brianvoe/gofakeit/v7"
)

var ErrInvalidOptionsType = errors.New("invalid options value type")

func transformOptions(options map[string]any) (*gofakeit.MapParams, error) {
	result := make(gofakeit.MapParams)
	for key, value := range options {
		if stringVal, ok := value.(string); ok {
			result[key] = []string{stringVal}

			continue
		}

		if stringArrayVal, ok := value.([]string); ok {
			result[key] = stringArrayVal

			continue
		}

		if intVal, ok := value.(int); ok {
			result[key] = []string{
				strconv.Itoa(intVal),
			}

			continue
		}

		if intVal, ok := value.(float64); ok {
			result[key] = []string{
				strconv.FormatFloat(intVal, 'g', -1, 64),
			}

			continue
		}

		return nil, ErrInvalidOptionsType
	}

	return &result, nil
}
