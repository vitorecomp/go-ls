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
		} else if outputParameters.Comparable {
			Comparable(file)
		} else {
			Default(file, outputParameters.ShowHidden, outputParameters.Color)
		}
	}
	if !outputParameters.List && !outputParameters.Comparable {
		fmt.Println()
	}
	defer wg.Done()
}
