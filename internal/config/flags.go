package config

import (
	"fmt"

	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/pflag"
)

func defineFlags(version string) *pflag.FlagSet {
	flags := pflag.NewFlagSet("uncors", pflag.ContinueOnError)
	flags.Usage = func() {
		tui.PrintLogo(flags.Output(), version)
		fmt.Fprintln(flags.Output(), "")
		fmt.Fprintln(flags.Output(), flags.FlagUsages())
	}
	flags.StringSliceP("to", "t", []string{}, "Target host with protocol for the resource to be proxied")
	flags.StringSliceP("from", "f", []string{}, "Local host with protocol for the resource from which proxying will take place") //nolint: lll
	flags.String("proxy", "", "HTTP/HTTPS proxy for requests to the real server (uses system proxy by default)")
	flags.StringP("config", "c", "", "Path to the configuration file")
	flags.Bool("interactive", true, "Run application in interactive TUI mode")

	return flags
}
