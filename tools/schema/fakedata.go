package main

import (
	"sort"

	"github.com/Jeffail/gabs"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/evg4b/uncors/pkg/fakedata"
	"github.com/samber/lo"
)

func generateFakeDataNodes() []*gabs.Container {
	items := lo.Filter(fakedata.GetTypes(), func(key string, _ int) bool {
		return key != "object" && key != "array"
	})

	sort.Strings(items)

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
				property := getPropertyBase(param.Type)
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

func getPropertyBase(typeDef string) *gabs.Container {
	object := gabs.New()

	switch typeDef {
	case "string":
		p(object, "type", "string")
	case "int":
		p(object, "type", "integer")
	case "uint":
		p(object, "type", "integer")
		p(object, "minimum", 0)
	case "float64":
		p(object, "type", "number")
	case "float":
		p(object, "type", "number")
	case "bool":
		p(object, "type", "boolean")
	case "[]string":
		p(object, "type", "array")
		p(object, "items.type", "string")
	default:
		panic("Unknown type: " + typeDef)
	}

	return object
}
