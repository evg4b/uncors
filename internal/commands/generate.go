package commands

import "context"

func generate(ctx context.Context, args []string, cwd string) {
	print("generate = args: ", args, " cwd: ", cwd)
}
