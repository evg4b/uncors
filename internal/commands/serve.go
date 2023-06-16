package commands

import "context"

func serve(ctx context.Context, args []string, cwd string) {
	print("serve = args: ", args, " cwd: ", cwd)
}
