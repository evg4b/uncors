package fakedata

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/samber/lo"
)

func getTypes() {
	for _, i2 := range lo.Keys(gofakeit.FuncLookups) {
		println(i2)
	}
}
