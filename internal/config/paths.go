package config

import (
	"os"
	"path/filepath"
	"strings"
)

// resolvePaths makes every path in the configuration absolute, once, so that
// handlers, validators and the HAR writer all see the same file.
//
// Three rules, and they are the ones tools like ESLint, Prettier and Docker
// Compose converge on:
//
//   - an absolute path is used as it is;
//   - "~" expands to the user's home directory, because home-relative paths in a
//     config file are a normal expectation and the shell cannot expand these
//     ones — they are inside the YAML, not on the command line;
//   - a relative path resolves against the directory holding the config file, so
//     that a config file is portable and does not depend on where uncors was
//     started. Without a config file, the working directory is the base.
func resolvePaths(cfg *UncorsConfig, configPath string) {
	base := "."
	if configPath != "" {
		base = filepath.Dir(configPath)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}

	resolve := func(path string) string {
		return resolvePath(path, base, home)
	}

	for index := range cfg.Mappings {
		mapping := &cfg.Mappings[index]

		mapping.HAR.File = resolve(mapping.HAR.File)

		for static := range len(mapping.Statics) {
			mapping.Statics[static].Dir = resolve(mapping.Statics[static].Dir)
		}

		for mock := range len(mapping.Mocks) {
			mapping.Mocks[mock].Response.File = resolve(mapping.Mocks[mock].Response.File)
		}

		for script := range len(mapping.Scripts) {
			mapping.Scripts[script].File = resolve(mapping.Scripts[script].File)
		}
	}
}

func resolvePath(path, base, home string) string {
	switch {
	case path == "":
		return path
	case path == "~", strings.HasPrefix(path, "~/"):
		if home == "" {
			return path
		}

		return filepath.Join(home, strings.TrimPrefix(path, "~"))
	case filepath.IsAbs(path):
		return path
	default:
		return filepath.Join(base, path)
	}
}
