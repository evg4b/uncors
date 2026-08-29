package config

import (
	"fmt"

	"github.com/spf13/pflag"
)

// Flags are the command line overrides applied on top of the configuration
// file. They are parsed once, at startup: a reload re-reads the file and
// re-applies the same flags rather than re-parsing the command line, so saving
// the config file cannot change what the flags mean.
type Flags struct {
	set *pflag.FlagSet
}

// DefineFlags registers the proxy flags on the given set.
func DefineFlags(set *pflag.FlagSet) *Flags {
	set.StringSliceP("to", "t", []string{}, "Target host with protocol for the resource to be proxied")
	set.StringSliceP("from", "f", []string{},
		"Local host with protocol for the resource from which proxying will take place")
	set.String("proxy", "", "HTTP/HTTPS proxy for requests to the real server (uses system proxy by default)")
	set.Bool("debug", false, "Write debug output to uncors.log")
	set.StringP("config", "c", "", "Path to the configuration file")
	set.Bool("interactive", true, "Render the terminal UI (falls back to plain output when stdout is not a terminal)")

	return &Flags{set: set}
}

// ParseFlags defines and parses the proxy flags from the given arguments.
func ParseFlags(args []string) (*Flags, error) {
	set := pflag.NewFlagSet("uncors", pflag.ContinueOnError)
	flags := DefineFlags(set)

	err := set.Parse(args)
	if err != nil {
		return nil, fmt.Errorf("failed parsing flags: %w", err)
	}

	return flags, nil
}

// ConfigPath is the config file the flags point at, empty when none was given.
func (f *Flags) ConfigPath() string {
	value, _ := f.set.GetString("config")

	return value
}
