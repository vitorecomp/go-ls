package output

import (
	"os"

	"github.com/vitorecomp/go-ls/pkg/cli"
)

func Output(files []os.FileInfo, arguments cli.CLI) {
	Default(files, arguments.ShowHidden)
}
