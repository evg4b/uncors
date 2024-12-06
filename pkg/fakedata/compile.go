package fakedata

import (
	"errors"
	"fmt"
	"sort"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/samber/lo"
)

var ErrUnknownType = errors.New("unknown type")

func (s *GoFakeItGenerator) compileInternal(faker *gofakeit.Faker, node *Node) (any, error) {
	switch node.Type {
	case "object":
		return s.compileToMap(node.Properties, faker)
	case "array":
		return s.compileToArray(node.Item, node.Count, faker)
	default:
		funcInfo := gofakeit.GetFuncLookup(node.Type)
		if funcInfo == nil {
			return nil, fmt.Errorf("incorrect fake function %s: %w", node.Type, ErrUnknownType)
		}

		options, err := transformOptions(node.Options)
		if err != nil {
			return nil, err
		}

		return funcInfo.Generate(faker, options, funcInfo)
	}
}

func (s *GoFakeItGenerator) compileToArray(item *Node, count int, faker *gofakeit.Faker) ([]any, error) {
	result := make([]any, 0, count)
	for range count {
		compiled, err := s.compileInternal(faker, item)
		if err != nil {
			return nil, err
		}
		result = append(result, compiled)
	}

	return result, nil
}

func (s *GoFakeItGenerator) compileToMap(properties map[string]Node, faker *gofakeit.Faker) (any, error) {
	result := make(map[string]any)
	keys := lo.Keys(properties)
	sort.Strings(keys) // it is important to sort keys to get the same result every time
	for _, property := range keys {
		node := properties[property]
		compiledValue, err := s.compileInternal(faker, &node)
		if err != nil {
			return nil, err
		}
		result[property] = compiledValue
	}

	return result, nil
}
