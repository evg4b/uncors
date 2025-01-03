package main

import (
	"os"
	"path/filepath"

	"github.com/Jeffail/gabs"
	"github.com/samber/lo"
)

func loadDefinitions() map[string]*gabs.Container {
	entries, err := os.ReadDir("definitions")
	requireNoError(err)

	files := lo.Filter(entries, func(entry os.DirEntry, _ int) bool {
		return !entry.IsDir()
	})

	storage := make(map[string]*gabs.Container, len(files))

	return lo.Reduce(files, func(storage map[string]*gabs.Container, entry os.DirEntry, _ int) map[string]*gabs.Container {
		refName := ref(entry.Name())
		storage[refName] = f(filepath.Join("definitions", entry.Name()))

		return storage
	}, storage)
}
