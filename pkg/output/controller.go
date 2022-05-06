package output

import (
	"github.com/fatih/color"
	"github.com/vitorecomp/go-ls/pkg/cli"
	"github.com/vitorecomp/go-ls/pkg/models"
)

func Output(files []models.File, arguments cli.CLI) {
	//TODO understand the auto flag
	showColor := arguments.Color == "always" || arguments.Color == "auto"
	terminal := color.New()
	if showColor {
		terminal.DisableColor()
	}
	if len(arguments.Format) != 0 {

	} else if arguments.List {
		List(files, arguments.ShowHidden, showColor)
	} else {
		Default(files, arguments.ShowHidden, showColor)
	}
}
