package fakedata

import (
	"errors"
	"reflect"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/mitchellh/mapstructure"
)

func (root *Node) Compile() (any, error) {
	faker := gofakeit.New(root.Seed)

	switch root.Type {
	case "object":
		return compileToMap(root.Properties, faker)
	case "array":
		return compileToArray(root.Items, faker)
	default:
		faker := gofakeit.New(root.Seed)
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

func (root *Node) compileInternal(faker *gofakeit.Faker) (any, error) {
	switch root.Type {
	case "object":
		return compileToMap(root.Properties, nil)
	case "array":
		return compileToArray(root.Items, nil)
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

		return nil, errors.New("invalid options value type")
	}

	return &result, nil
}

func compileToArray(items []Node, faker *gofakeit.Faker) ([]any, error) {
	result := make([]any, 0, len(items))
	for _, value := range items {
		compiledValue, err := value.compileInternal(faker)
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

var rootNodeType = reflect.TypeOf(Node{})

func RootNodeDecodeHook() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, rawData interface{}) (interface{}, error) {
		return rawData, nil
	}
}
