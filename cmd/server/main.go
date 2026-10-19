// Command server starts the standalone go-ls HTTP server.
package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/vitorecomp/go-ls/pkg/cli"
)

func main() {
	var serveCmd cli.ServeCmd
	ctx := kong.Parse(&serveCmd,
		kong.Name("go-ls-server"),
		kong.Description("Standalone HTTP server for storing and comparing go-ls directory scans."),
		kong.UsageOnError(),
	)

	if err := serveCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "go-ls-server: %v\n", err)
		ctx.Exit(1)
	}
}
