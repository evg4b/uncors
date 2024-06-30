package fakedata

import (
	"sync"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/samber/lo"
)

var initPackage = sync.OnceFunc(func() {
	gofakeit.RemoveFuncLookup("csv")
	gofakeit.RemoveFuncLookup("xml")
	gofakeit.RemoveFuncLookup("json")
})

func GetTypes() []string {
	initPackage()

	types := lo.Keys(gofakeit.FuncLookups)
	types = append(types, "object")
	types = append(types, "array")

	return types
}
