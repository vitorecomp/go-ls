package output

import (
	"github.com/fatih/color"
	"github.com/vitorecomp/go-ls/pkg/cli"
	"github.com/vitorecomp/go-ls/pkg/models"
)

func Default(file models.File, showHidden bool, noColor bool) {
	terminal := color.New()

	if file.FileInfo.Name()[0] != '.' || showHidden {
		if file.FileInfo.IsDir() {
			terminal = color.New(color.Bold, color.FgBlue)
		} else {
			terminal = color.New()
		}
		terminal.Printf(" %s ", file.FileInfo.Name())
	}
}

func List(file models.File, showHidden bool, noColor bool, hash bool) {
	terminal := color.New()

	if file.FileInfo.Name()[0] != '.' || showHidden {
		file.FillOwnerAndGroup()
		terminal = color.New()

		terminal.Printf("%s %s %s %4d %s",
			file.FileInfo.Mode(),
			file.Owner.Username,
			file.Group.Name,
			file.FileInfo.Size(),
			file.FileInfo.ModTime().Format(cli.Constants.DateFormat),
		)

		if hash {
			terminal.Printf(" %s", file.Hash)
		}

		if file.FileInfo.IsDir() {
			terminal = color.New(color.Bold, color.FgBlue)
		}
		terminal.Printf(" %s", file.FileInfo.Name())
		terminal.Println()
	}
}
