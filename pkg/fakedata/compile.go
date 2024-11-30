package fakedata

import (
	"errors"
	"fmt"
	"sort"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/samber/lo"
)

var ErrUnknownType = errors.New("unknown type")

func (root *Node) Compile(seed uint64) (any, error) {
	initPackage()

	return root.compileInternal(gofakeit.New(seed))
}

func (root *Node) compileInternal(faker *gofakeit.Faker) (any, error) {
	switch root.Type {
	case "object":
		return compileToMap(root.Properties, faker)
	case "array":
		return compileToArray(root.Item, root.Count, faker)
	default:
		funcInfo := gofakeit.GetFuncLookup(root.Type)
		if funcInfo == nil {
			return nil, fmt.Errorf("incorrect fake function %s: %w", root.Type, ErrUnknownType)
		}

		options, err := transformOptions(root.Options)
		if err != nil {
			return nil, err
		}

		return funcInfo.Generate(faker, options, funcInfo)
	}
}

func compileToArray(item *Node, count int, faker *gofakeit.Faker) ([]any, error) {
	result := make([]any, 0, count)
	for range count {
		compiled, err := item.compileInternal(faker)
		if err != nil {
			return nil, err
		}
		result = append(result, compiled)
	}

	return result, nil
}

func compileToMap(properties map[string]Node, faker *gofakeit.Faker) (any, error) {
	result := make(map[string]any)
	keys := lo.Keys(properties)
	sort.Strings(keys) // it is important to sort keys to get the same result every time
	for _, property := range keys {
		node := properties[property]
		compiledValue, err := node.compileInternal(faker)
		if err != nil {
			return nil, err
		}
		result[property] = compiledValue
	}

	return result, nil
}
