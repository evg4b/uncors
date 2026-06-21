package di

import "sync"

type factory[T any] struct {
	cache   *T
	factory func() *T
	sync.Mutex
}

func (f *factory[T]) GetOrBuild() *T {
	f.Lock()
	defer f.Unlock()

	if f.cache != nil {
		return f.cache
	}

	f.cache = f.factory()
	return f.cache
}

func newFactory[T any](new func() *T) factory[T] {
	return factory[T]{factory: new}
}
