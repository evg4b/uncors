package main

import (
	"github.com/Jeffail/gabs"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/evg4b/uncors/pkg/fakedata"
	"github.com/samber/lo"
)

func generateFakeDataNodes() []*gabs.Container {
	items := lo.Filter(fakedata.GetTypes(), func(key string, _ int) bool {
		return key != "object" && key != "array"
	})

	array := lo.Map(items, func(key string, _ int) *gabs.Container {
		info := gofakeit.GetFuncLookup(key)
		if info == nil {
			panic("Unknown type: " + key)
		}

		item := o()
		p(item, "title", info.Display)
		p(item, "description", info.Description)
		p(item, "properties.type.const", key)
		p(item, "required", []string{"type"})
		p(item, "examples", []string{info.Example})

		if len(info.Params) > 0 {
			options := o()

			for _, param := range info.Params {
				property := gabs.New()
				p(property, "type", getSchemaType(param.Type))
				p(property, "title", param.Display)
				p(property, "description", param.Description)
				p(property, "default", param.Default)

				p(options, "properties."+param.Field, property.Data())
			}

			p(item, "properties.options", options.Data())
		}

		return item
	})

	return append(
		array,
		f("./fakedata/object.json"),
		f("./fakedata/array.json"),
	)
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
	case "float":
		return "number" // nolint: goconst
	case "bool":
		return "boolean"
	default:
		return "string"
	}
}
