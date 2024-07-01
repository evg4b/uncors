package main

import (
	"encoding/json"
	"os"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/evg4b/uncors/pkg/fakedata"
	"github.com/samber/lo"
)

type SchemaNode struct {
	Name                 string                `json:"name,omitempty"`
	Description          string                `json:"description,omitempty"`
	Type                 string                `json:"type,omitempty"`
	Minimum              int                   `json:"minimum,omitempty"`
	Const                string                `json:"const,omitempty"`
	Default              string                `json:"default,omitempty"`
	AdditionalProperties bool                  `json:"additionalProperties"`
	Ref                  string                `json:"$ref,omitempty"`
	Properties           map[string]SchemaNode `json:"properties,omitempty"`
	Required             []string              `json:"required,omitempty"`
}

func generateSchemeData() {
	var nodes []SchemaNode //nolint:prealloc

	for _, key := range fakedata.GetTypes() {
		if key == "object" {
			nodes = append(nodes, SchemaNode{
				Type: "object",
				Properties: map[string]SchemaNode{
					"properties": {
						Ref: "#/definitions/FakeDataNode",
					},
					"type": {
						Const: "object",
					},
				},
				Required: []string{"type"},
			})

			continue
		}

		if key == "array" {
			nodes = append(nodes, SchemaNode{
				Type: "object",
				Properties: map[string]SchemaNode{
					"items": {
						Ref: "#/definitions/FakeDataNode",
					},
					"type": {
						Const: "array",
					},
					"count": {
						Type:    "integer",
						Minimum: 0,
					},
				},
				Required: []string{"type"},
			})

			continue
		}

		info := gofakeit.GetFuncLookup(key)
		if info == nil {
			panic("Unknown type: " + key)
		}

		schemaNode := SchemaNode{
			Name:                 info.Display,
			Description:          info.Description,
			Type:                 "object",
			AdditionalProperties: false,
			Properties: map[string]SchemaNode{
				"type": {
					Const: key,
				},
			},
		}

		if info.Params != nil && len(info.Params) > 0 {
			params := map[string]SchemaNode{}

			lo.ForEach(info.Params, func(v gofakeit.Param, _ int) {
				params[v.Field] = SchemaNode{
					Type:        getSchemaType(v.Type),
					Default:     v.Default,
					Description: v.Description,
					Name:        v.Display,
				}
			})

			schemaNode.Properties["options"] = SchemaNode{
				Type:                 "object",
				AdditionalProperties: false,
				Properties:           params,
			}
		}

		nodes = append(nodes, schemaNode)
	}

	file, err := os.Create("tools/fakedata/scheme.json")
	if err != nil {
		panic(err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			panic(err)
		}
	}(file)

	if err = json.NewEncoder(file).Encode(nodes); err != nil {
		panic(err)
	}
}

func getSchemaType(typeDef string) string {
	switch typeDef {
	case "string": // nolint: goconst
		return "string"
	case "int":
		return "number" // nolint: goconst
	case "uint":
		return "number" // nolint: goconst
	case "float64":
		return "number" // nolint: goconst
	case "bool":
		return "boolean"
	default:
		return "string"
	}
}
