package commands

import "context"

type Command = func(ctx context.Context, args []string, cwd string)

var commands = map[string]Command{
	"generate": generate,
	"version":  version,
	"serve":    serve,
}

const defaultCommand = "serve"

func Run(ctx context.Context, args []string, cwd string) {
	if command, ok := commands[args[1]]; ok {
		command(ctx, args, cwd)
	} else {
		commands[defaultCommand](ctx, args, cwd)
	}
}
