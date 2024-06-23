package fakedata

import (
	"errors"
	"github.com/brianvoe/gofakeit/v7"
)

func (root *Node) Compile() (any, error) {
	faker := gofakeit.New(root.Seed)

	return root.compileInternal(faker)
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
			return nil, errors.New("unknown type: " + root.Type)
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
		compiledValue, err := item.compileInternal(faker)
		if err != nil {
			return nil, err
		}
		result = append(result, compiledValue)
	}

	return result, nil
}

func compileToMap(properties map[string]Node, faker *gofakeit.Faker) (map[string]any, error) {
	result := make(map[string]any)
	for key, value := range properties {
		compiledValue, err := value.compileInternal(faker)
		if err != nil {
			return nil, err
		}
		result[key] = compiledValue
	}

	return result, nil
}
