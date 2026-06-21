package di

import "sync"

type factory[T any] struct {
	once sync.Once

	cache   T
	factory func() T
}

func (f *factory[T]) GetOrBuild() T {
	f.once.Do(func() {
		f.cache = f.factory()
	})

	return f.cache
}

func newFactory[T any](factoryFunc func() T) factory[T] {
	return factory[T]{factory: factoryFunc}
}

type factory1[T any, D comparable] struct {
	once sync.Once

	cache   T
	factory func(arg D) T
}

func (f *factory1[T, D]) GetOrBuild(arg D) T {
	f.once.Do(func() {
		f.cache = f.factory(arg)
	})

	return f.cache
}

func newFactory1[T any, D comparable](factoryFunc func(arg D) T) factory1[T, D] {
	return factory1[T, D]{factory: factoryFunc}
}
