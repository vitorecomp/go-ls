package output

import (
	"fmt"
	"sync"

	"github.com/fatih/color"
	"github.com/vitorecomp/go-ls/pkg/cli"
	"github.com/vitorecomp/go-ls/pkg/models"
)

func Output(wg *sync.WaitGroup, outputChannel chan models.File, arguments cli.CLI) {
	showColor := arguments.Color == "always" || arguments.Color == "auto"
	terminal := color.New()
	for file := range outputChannel {
		if showColor {
			terminal.DisableColor()
		}
		if arguments.List {
			List(file, arguments.ShowHidden, showColor, arguments.Hash)
		} else {
			Default(file, arguments.ShowHidden, showColor)
		}
	}
	if !arguments.List {
		fmt.Println()
	}
	defer wg.Done()
}
