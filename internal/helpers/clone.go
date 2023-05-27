package helpers

import "github.com/samber/lo"

func CloneMap[K comparable, V any](data map[K]V) map[K]V {
	cloned := make(map[K]V, len(data))
	for key, value := range data {
		if cloneable, ok := any(value).(lo.Clonable[V]); ok {
			cloned[key] = cloneable.Clone()
		} else {
			cloned[key] = value
		}
	}

	return cloned
}
