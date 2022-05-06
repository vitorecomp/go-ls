package output

import (
	"github.com/fatih/color"
	"github.com/vitorecomp/go-ls/pkg/cli"
	"github.com/vitorecomp/go-ls/pkg/models"
)

func Default(files []models.File, showHidden bool, noColor bool) {
	terminal := color.New()

	for _, file := range files {
		if file.FileInfo.Name()[0] != '.' || showHidden {
			if file.FileInfo.IsDir() {
				terminal = color.New(color.Bold, color.FgBlue)
			} else {
				terminal = color.New()
			}
			terminal.Printf(" %s ", file.FileInfo.Name())
		}
	}
	terminal.Printf("\n")
}

func List(files []models.File, showHidden bool, noColor bool) {
	terminal := color.New()

	for _, file := range files {
		if file.FileInfo.Name()[0] != '.' || showHidden {
			file.FillOwnerAndGroup()
			terminal = color.New()

			terminal.Printf("%s %s %s %4d %s",
				file.FileInfo.Mode(),
				file.Owner.Username,
				file.Group.Name,
				file.FileInfo.Size(),
				file.FileInfo.ModTime().Format(cli.Constants.DateFormat))
			if file.FileInfo.IsDir() {
				terminal = color.New(color.Bold, color.FgBlue)
			}
			terminal.Printf(" %s\n", file.FileInfo.Name())
		}
	}
}
