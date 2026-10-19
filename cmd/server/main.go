// Command server starts the standalone go-ls HTTP server.
package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/vitorecomp/go-ls/pkg/cli"
)

func main() {
	if cli.HasVersionFlag(os.Args[1:]) {
		fmt.Printf("go-ls-server v%s\n", cli.Version)
		return
	}

	var app struct {
		Version      kong.VersionFlag `help:"Output version information and exit." name:"version" short:"v"`
		cli.ServeCmd `embed:""`
	}
	ctx := kong.Parse(&app,
		kong.Name("go-ls-server"),
		kong.Description("Standalone HTTP server for storing and comparing go-ls directory scans."),
		kong.Vars{"version": fmt.Sprintf("go-ls-server v%s", cli.Version)},
		kong.UsageOnError(),
	)

	if err := app.ServeCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "go-ls-server: %v\n", err)
		ctx.Exit(1)
	}
}
