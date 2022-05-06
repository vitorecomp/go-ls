package main

import (
	"os"

	"github.com/alecthomas/kong"
	"github.com/vitorecomp/go-ls/pkg/cli"
	"github.com/vitorecomp/go-ls/pkg/looker"
	"github.com/vitorecomp/go-ls/pkg/output"
)

func main() {
	kong.Parse(&cli.Arguments)

	//look if has a path argument, if not
	files := []os.FileInfo{}
	for _, path := range cli.Arguments.Paths {
		pathFiles := looker.Look(path, cli.Arguments.Recursive)
		files = append(files, pathFiles...)
	}

	output.Output(files, cli.Arguments)
}