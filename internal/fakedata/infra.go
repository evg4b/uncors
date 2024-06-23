package fakedata

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/samber/lo"
)

func GetTypes() []string {
	return lo.Keys(gofakeit.FuncLookups)
}
