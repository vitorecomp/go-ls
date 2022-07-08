package output

import (
	"fmt"
	"sync"

	"github.com/fatih/color"
	"github.com/vitorecomp/go-ls/pkg/models"
)

func Output(wg *sync.WaitGroup, outputChannel chan models.File, outputParameters models.OutputParameters) {

	terminal := color.New()
	for file := range outputChannel {
		if outputParameters.Color {
			terminal.DisableColor()
		}
		if outputParameters.List {
			List(file, outputParameters.ShowHidden, outputParameters.Color, outputParameters.Hash)
		} else {
			Default(file, outputParameters.ShowHidden, outputParameters.Color)
		}
	}
	if !outputParameters.List {
		fmt.Println()
	}
	defer wg.Done()
}
