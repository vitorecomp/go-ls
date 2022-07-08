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
	//closing the
	defer wg.Wait()

	kong.Parse(&cli.Arguments)

	//generate lookers modifiers
	// that will define all the data that will be colocted by the looker
	lookParamters := models.LookParameters{
		Recursive: cli.Arguments.Recursive,
		Hash:      cli.Arguments.Hash,
	}

	//generate output modifiers
	// that will define the way that the data will be showed

	//create output channel
	outputChannel := make(chan models.File, 100)
	defer close(outputChannel)

	wg.Add(1)
	go output.Output(&wg, outputChannel, cli.Arguments)

	//create the routine pool

	//look if has a path argument, if not
	for _, path := range cli.Arguments.Paths {
		looker.Look(path, lookParamters, outputChannel)
	}

}
