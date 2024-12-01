package fakedata

import (
	"github.com/brianvoe/gofakeit/v7"
)

type Generator interface {
	Generate(node *Node, seed uint64) (any, error)
}

type GoFakeItGenerator struct{}

func NewGoFakeItGenerator() *GoFakeItGenerator {
	initPackage()

	return &GoFakeItGenerator{}
}

func (s *GoFakeItGenerator) Generate(node *Node, seed uint64) (any, error) {
	return s.compileInternal(gofakeit.New(seed), node)
}
