// Package cli is the uncors command tree: it resolves the command named on the
// command line, parses its flags and runs it. Every command goes through the
// same path, so help, --version and flag error handling exist once rather than
// once per command.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"text/tabwriter"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
)

const (
	exitOK    = 0
	exitError = 1
)

// Env is what a command needs from the process it runs in.
type Env struct {
	Fs      afero.Fs
	Stdout  *os.File
	Console contracts.Output
	Version string
}

// Command is one entry in the command tree. The root command has an empty name.
type Command struct {
	Name  string
	Short string
	Flags func(set *pflag.FlagSet)
	Run   func(ctx context.Context, env Env, set *pflag.FlagSet) error
}

// Execute resolves and runs the command named by args (excluding the program
// name) and returns the process exit code.
func Execute(ctx context.Context, env Env, args []string) int {
	defer PanicInterceptor(func(value any) {
		env.Console.Error(value)
		log.Fatalf("Caught panic: %v", value)
	})

	commands := []Command{proxyCommand(), generateCertsCommand()}

	command, rest := resolve(commands, args)

	set := pflag.NewFlagSet(commandLine(command), pflag.ContinueOnError)
	set.SortFlags = false
	set.Usage = func() { usage(env, commands, command, set) }

	showVersion := set.BoolP("version", "v", false, "Print the uncors version and exit")

	if command.Flags != nil {
		command.Flags(set)
	}

	err := set.Parse(rest)
	if err != nil {
		// A help request is not a failure, and it is handled here rather than
		// in every command.
		if errors.Is(err, pflag.ErrHelp) {
			return exitOK
		}

		return report(env, err)
	}

	if *showVersion {
		fmt.Fprintln(env.Console, env.Version)

		return exitOK
	}

	err = command.Run(ctx, env, set)
	if err != nil {
		return report(env, err)
	}

	return exitOK
}

func resolve(commands []Command, args []string) (Command, []string) {
	if len(args) > 0 {
		for _, command := range commands {
			if command.Name != "" && command.Name == args[0] {
				return command, args[1:]
			}
		}
	}

	return commands[0], args
}

func commandLine(command Command) string {
	if command.Name == "" {
		return "uncors"
	}

	return "uncors " + command.Name
}

func usage(env Env, commands []Command, current Command, set *pflag.FlagSet) {
	tui.PrintLogo(env.Console, env.Version)

	fmt.Fprintf(env.Console, "Usage:\n  %s [flags]\n\n", commandLine(current))

	if current.Name == "" {
		fmt.Fprintln(env.Console, "Commands:")
		printCommands(env.Console, commands)
		fmt.Fprintln(env.Console)
	}

	fmt.Fprintln(env.Console, "Flags:")
	fmt.Fprint(env.Console, set.FlagUsages())
}

func printCommands(out io.Writer, commands []Command) {
	writer := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0) //nolint:mnd // tabwriter layout

	for _, command := range commands {
		if command.Name == "" {
			continue
		}

		fmt.Fprintf(writer, "  %s\t%s\n", command.Name, command.Short)
	}

	_ = writer.Flush()
}

func report(env Env, err error) int {
	slog.Error("command failed", "err", err)
	env.Console.Error(err)

	return exitError
}
