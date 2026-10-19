// Command client runs the go-ls client CLI.
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
			fmt.Printf("go-ls-client v%s\n", cli.Version)
			return
		}
	}

	var app cli.CLI
	parser, err := kong.New(&app,
		kong.Name("go-ls-client"),
		kong.Description("Client CLI for scanning directories and querying the go-ls server."),
		kong.UsageOnError(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "go-ls-client: %v\n", err)
		os.Exit(1)
	}

	ctx, err := parser.Parse(cli.NormalizeArgs(rawArgs))
	parser.FatalIfErrorf(err)

	err = ctx.Run()
	parser.FatalIfErrorf(err)
}
