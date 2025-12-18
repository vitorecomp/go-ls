package linux

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/alecthomas/kong"
	"github.com/vitorecomp/go-ls/pkg/cli"
	"github.com/vitorecomp/go-ls/pkg/looker"
	"github.com/vitorecomp/go-ls/pkg/models"
	"github.com/vitorecomp/go-ls/pkg/output"
)

func run(wg *sync.WaitGroup, lookParameters models.LookParameters, outputChannel chan models.File) {

	//run the paths
	for _, path := range cli.Arguments.Paths {
		fileInfo, err := os.Stat(path)
		if err != nil {
			log.Fatal(err)
		}

		if fileInfo.IsDir() {
			basePath := filepath.Dir(path)
			workingFile := models.File{
				FileInfo: fileInfo,
				BasePath: basePath,
			}
			wg.Add(1)
			go looker.Look(wg, path, lookParameters, outputChannel)
			outputChannel <- workingFile

		} else {
			log.Fatal("Path params is not a dir:" + path)
		}
	}
	wg.Wait()
	close(outputChannel)
}
func Main() {
	var wg sync.WaitGroup
	//closing the wait group

	kong.Parse(&cli.Arguments)

	//generate lookers modifiers
	// that will define if data that will be collected by the looker
	lookParameters := models.BuildLookParameters(cli.Arguments)

	//create output channel
	outputChannel := make(models.OutputChan, 1000)
	go run(&wg, lookParameters, outputChannel)
	outputParameters := models.BuildOutputParameters(cli.Arguments)
	output.Output(outputChannel, outputParameters)

}
