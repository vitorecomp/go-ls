// Command go-ls is a unified client-server CLI for concurrent directory scanning,
// hashing, remote storage, diffing, and duplicate detection.
package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/vitorecomp/go-ls/pkg/cli"
)

func main() {
	rawArgs := os.Args[1:]
	for _, a := range rawArgs {
		if a == "--version" {
			fmt.Printf("go-ls v%s\n", cli.Version)
			return
		}
	}

	var app cli.CLI
	parser, err := kong.New(&app,
		kong.Name("go-ls"),
		kong.Description("Client-server directory listing, hashing, comparison, and backup verification tool."),
		kong.UsageOnError(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "go-ls: %v\n", err)
		os.Exit(1)
	}

	ctx, err := parser.Parse(cli.NormalizeArgs(rawArgs))
	parser.FatalIfErrorf(err)

	err = ctx.Run()
	parser.FatalIfErrorf(err)
}
