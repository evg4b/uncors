package config

import (
	"github.com/spf13/pflag"
)

func defineFlags(usage Usage) *pflag.FlagSet {
	flags := pflag.NewFlagSet("uncors", pflag.ContinueOnError)
	flags.Usage = func() {
		if usage != nil {
			usage(flags)
		}
	}
	flags.StringSliceP("to", "t", []string{}, "Target host with protocol for the resource to be proxied")
	flags.StringSliceP("from", "f", []string{},
		"Local host with protocol for the resource from which proxying will take place")
	flags.String("proxy", "", "HTTP/HTTPS proxy for requests to the real server (uses system proxy by default)")
	flags.StringP("config", "c", "", "Path to the configuration file (defaults to .uncors.yaml or .uncors.yml)")
	flags.Bool("interactive", true, "Run application in interactive TUI mode")
	flags.BoolP("version", "v", false, "Print the version and exit")

	return flags
}
