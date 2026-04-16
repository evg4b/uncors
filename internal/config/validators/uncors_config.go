package validators

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/spf13/afero"
)

// ValidateConfig validates the full uncors configuration and returns a combined
// error listing all validation failures. Returns nil if the config is valid.
func ValidateConfig(cfg *config.UncorsConfig, fs afero.Fs) error {
	var errs Errors

	if len(cfg.Mappings) == 0 {
		errs.add("mappings must not be empty")
		return errs
	}

	for i, mapping := range cfg.Mappings {
		ValidateMapping(joinPath("mappings", index(i)), mapping, fs, &errs)
	}

	ValidateProxy("proxy", cfg.Proxy, &errs)
	ValidateCacheConfig("cache-config", cfg.CacheConfig, &errs)

	if errs.HasAny() {
		return errs
	}

	return nil
}
