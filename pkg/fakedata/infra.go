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
	gofakeit.RemoveFuncLookup("person")
	gofakeit.RemoveFuncLookup("teams")
	gofakeit.RemoveFuncLookup("car")
	gofakeit.RemoveFuncLookup("movie")
	gofakeit.RemoveFuncLookup("product")
	gofakeit.RemoveFuncLookup("creditcard")
	gofakeit.RemoveFuncLookup("address")
	gofakeit.RemoveFuncLookup("email_text")
})

func GetTypes() []string {
	initPackage()

	types := lo.Keys(gofakeit.FuncLookups)
	types = append(types, "object")
	types = append(types, "array")

	return types
}
