package fakedata

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/samber/lo"
)

func init() {
	gofakeit.RemoveFuncLookup("csv")
	gofakeit.RemoveFuncLookup("xml")
	gofakeit.RemoveFuncLookup("json")
}

func GetTypes() []string {
	types := lo.Keys(gofakeit.FuncLookups)
	types = append(types, "object")
	types = append(types, "array")

	return types
}
