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
	var chanStack models.Stack
	//closing the
	defer wg.Wait()

	kong.Parse(&cli.Arguments)
	//teste de commit

	//generate lookers modifiers
	// that will define all the data that will be collected by the looker
	lookParameters := models.BuildLookParameters(cli.Arguments)
	outputParameters := models.BuildOutputParameters(cli.Arguments)

	//creating the channels (hash, sort, output)

	//create output channel
	outputChannel := make(models.OutputChan, 1000)
	chanStack = chanStack.Push(outputChannel)
	defer close(outputChannel)

	//TODO add here the sort channel if setted

	//create output hash channel
	if lookParameters.Hash {
		hashChannel := make(models.OutputChan, 1000000)
		defer close(hashChannel)
		chanStack = chanStack.Push(hashChannel)
	}

	//run the paths
	for _, path := range cli.Arguments.Paths {
		wg.Add(1)
		go looker.Look(&wg, path, lookParameters, chanStack.Head())
	}

	//add the channels consumers
	if lookParameters.Hash {
		var inputChannel models.OutputChan
		chanStack, inputChannel = chanStack.Pop()
		wg.Add(1)
		go looker.Hash(&wg, inputChannel, chanStack.Head(), lookParameters)
	}

	//TODO add here the sort consumer if setted

	wg.Add(1)
	go output.Output(&wg, chanStack.Head(), outputParameters)
	//create the routine pool
}
