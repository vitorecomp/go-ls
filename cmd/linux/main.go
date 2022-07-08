package linux

import (
	"sync"

	"github.com/alecthomas/kong"
	"github.com/vitorecomp/go-ls/pkg/cli"
	"github.com/vitorecomp/go-ls/pkg/looker"
	"github.com/vitorecomp/go-ls/pkg/models"
	"github.com/vitorecomp/go-ls/pkg/output"
)

func Main() {
	var wg sync.WaitGroup

	kong.Parse(&cli.Arguments)

	//generate lookers modifiers
	lookParamters := models.LookParameters{
		Recursive: cli.Arguments.Recursive,
		Hash:      cli.Arguments.Hash,
	}

	//generate output modifiers

	//create output channel
	outputChannel := make(chan models.File, 100)
	wg.Add(1)
	go output.Output(&wg, outputChannel, cli.Arguments)

	//create the routine pool

	//look if has a path argument, if not
	for _, path := range cli.Arguments.Paths {
		looker.Look(path, lookParamters, outputChannel)
	}
	close(outputChannel)
	wg.Wait()
}
