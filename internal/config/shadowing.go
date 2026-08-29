package config

import (
	"log/slog"
	"strings"
)

// warnAboutShadowedRoutes reports routes that can never be reached.
//
// Route precedence puts mocks, scripts and rewrites ahead of static mounts, so a
// static mount can no longer hide them. What it can still hide is another static
// mount underneath it: the first prefix that matches wins, so `/` mounted before
// `/assets` claims everything.
func warnAboutShadowedRoutes(cfg *UncorsConfig) {
	for _, mapping := range cfg.Mappings {
		for index, static := range mapping.Statics {
			for _, later := range mapping.Statics[index+1:] {
				if shadows(static.Path, later.Path) {
					slog.Warn("a static mount is unreachable: an earlier mount already covers it",
						"host", mapping.From.String(),
						"unreachable", later.Path,
						"covered_by", static.Path)
				}
			}
		}
	}
}

// shadows reports whether a mount at prefix claims every path a mount at other
// would serve.
func shadows(prefix, other string) bool {
	prefix = strings.TrimSuffix(prefix, "/")
	other = strings.TrimSuffix(other, "/")

	if prefix == "" {
		return true
	}

	return other == prefix || strings.HasPrefix(other, prefix+"/")
}
