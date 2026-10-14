// Command go-ls is a concurrent file listing and hashing CLI utility written in Go.
package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/vitorecomp/go-ls/pkg/cli"
	"github.com/vitorecomp/go-ls/pkg/looker"
	"github.com/vitorecomp/go-ls/pkg/models"
	"github.com/vitorecomp/go-ls/pkg/output"
)

const defaultChannelBuffer = 1024

func main() {
	var args cli.CLI
	kong.Parse(&args,
		kong.Name("go-ls"),
		kong.Description("High-performance concurrent directory listing, hashing, and comparison tool."),
		kong.UsageOnError(),
	)

	if args.Version {
		fmt.Printf("go-ls v%s\n", cli.Version)
		return
	}

	lookOpts := args.ToLookOptions()
	outputOpts := args.ToOutputOptions()

	outputChannel := make(models.OutputChan, defaultChannelBuffer)
	go looker.Run(args.Paths, lookOpts, outputChannel, os.Stderr)

	output.Output(outputChannel, outputOpts)
}
