package validators

import (
	"net/url"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/urlparser"
)

// ValidateHostsFileEntries checks if all hosts from mappings are present in the hosts file.
// If a host is not found in the hosts file, it logs a warning.
func ValidateHostsFileEntries(cfg *config.UncorsConfig) {
	hosts, err := helpers.ReadHostsFile()
	if err != nil {
		log.Warnf("Failed to read hosts file: %v. Skipping hosts validation.", err)
		return
	}

	for _, mapping := range cfg.Mappings {
		parsedURL, err := urlparser.Parse(mapping.From)
		if err != nil {
			log.Warnf("Failed to parse 'from' URL '%s': %v", mapping.From, err)
			continue
		}

		hostname := getHostnameFromURL(parsedURL)

		// Skip wildcards - they can't be validated against hosts file
		if containsWildcard(hostname) {
			continue
		}

		if !helpers.IsHostInHostsFile(hostname, hosts) {
			log.Warnf(
				"Host '%s' from mapping '%s' -> '%s' is not found in hosts file or does not point to localhost. "+
					"Add '127.0.0.1 %s' to %s for proper functionality.",
				hostname, mapping.From, mapping.To, hostname, helpers.GetHostsFilePath(),
			)
		}
	}
}

// getHostnameFromURL extracts the hostname from a URL, without port.
func getHostnameFromURL(u *url.URL) string {
	host := u.Hostname()
	if host == "" {
		host = u.Host
	}

	return host
}

// containsWildcard checks if a hostname contains wildcard characters.
func containsWildcard(hostname string) bool {
	return len(hostname) > 0 && hostname[0] == '*'
}
