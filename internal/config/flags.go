package config

import (
	"github.com/spf13/pflag"
)

// UsageRenderer draws the --help output for a flag set. Rendering it is a
// presentation decision, so the caller supplies one rather than this package
// reaching for the console.
type UsageRenderer func(flags *pflag.FlagSet, version string)

// Option configures how the configuration is loaded.
type Option func(*loadOptions)

type loadOptions struct {
	usage UsageRenderer
}

// WithUsage sets the renderer used for --help. Without it, pflag's own default
// usage output is used.
func WithUsage(usage UsageRenderer) Option {
	return func(o *loadOptions) {
		o.usage = usage
	}
}

func defineFlags(version string, opts loadOptions) *pflag.FlagSet {
	flags := pflag.NewFlagSet("uncors", pflag.ContinueOnError)

	if opts.usage != nil {
		flags.Usage = func() { opts.usage(flags, version) }
	}

	flags.StringSliceP("to", "t", []string{}, "Target host with protocol for the resource to be proxied")
	flags.StringSliceP("from", "f", []string{}, "Local host with protocol for the resource from which proxying will take place") //nolint: lll
	flags.String("proxy", "", "HTTP/HTTPS proxy for requests to the real server (uses system proxy by default)")
	flags.StringP("config", "c", "", "Path to the configuration file")
	flags.Bool("interactive", true, "Run application in interactive TUI mode")
	flags.BoolP("version", "v", false, "Print the version and exit")

	return flags
}
