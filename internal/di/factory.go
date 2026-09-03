package di

import "sync"

type factory[T any] struct {
	once sync.Once

	built   bool
	cache   T
	factory func() T
}

func (f *factory[T]) GetOrBuild() T {
	f.once.Do(func() {
		f.cache = f.factory()
		f.built = true
	})

	return f.cache
}

// Built reports whether the value has been created. Replacing a factory after
// that point cannot take effect, so callers use this to fail loudly instead of
// silently keeping the old value.
func (f *factory[T]) Built() bool {
	return f.built
}

func newFactory[T any](factoryFunc func() T) factory[T] {
	return factory[T]{factory: factoryFunc}
}
