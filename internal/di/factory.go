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
