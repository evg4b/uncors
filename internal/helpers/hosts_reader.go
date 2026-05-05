package helpers

import (
	"bufio"
	"net"
	"runtime"
	"strings"

	"github.com/spf13/afero"
)

const minHostsFileFields = 2 // Minimum fields in hosts file: IP and at least one hostname

// GetHostsFilePath returns the path to the system hosts file based on the operating system.
func GetHostsFilePath() string {
	if runtime.GOOS == "windows" {
		return `C:\Windows\System32\drivers\etc\hosts`
	}

	return "/etc/hosts"
}

// HostsEntry represents a single entry in the hosts file.
type HostsEntry struct {
	IP        string
	Hostnames []string
}

// ReadHostsFile reads and parses the system hosts file using the provided filesystem.
// It returns a map where the key is the hostname and the value is the IP address.
func ReadHostsFile(fs afero.Fs) (map[string]string, error) {
	hostsPath := GetHostsFilePath()
	file, err := fs.Open(hostsPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hosts := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split by whitespace
		fields := strings.Fields(line)
		if len(fields) < minHostsFileFields {
			continue
		}

		ip := fields[0]
		// Add all hostnames from this line
		for _, hostname := range fields[1:] {
			hosts[hostname] = ip
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return hosts, nil
}

// IsLocalhost checks if the given IP address is localhost (127.0.0.1, ::1, or any loopback address).
func IsLocalhost(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	return parsedIP.IsLoopback()
}

// IsHostInHostsFile checks if a hostname is defined in the hosts file and points to localhost.
func IsHostInHostsFile(hostname string, hosts map[string]string) bool {
	ip, exists := hosts[hostname]
	if !exists {
		return false
	}

	return IsLocalhost(ip)
}
