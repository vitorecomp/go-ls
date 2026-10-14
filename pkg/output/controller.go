package output

import (
	"fmt"
	"io"
	"os"

	"github.com/vitorecomp/go-ls/pkg/models"
)

// Output consumes discovered files from outputChannel and prints them to os.Stdout
// according to outputParameters.
func Output(outputChannel <-chan models.File, outputParameters models.OutputParameters) {
	WriteOutput(os.Stdout, outputChannel, outputParameters)
}

// WriteOutput consumes discovered files from outputChannel and writes formatted output to w.
func WriteOutput(w io.Writer, outputChannel <-chan models.File, opts models.OutputOptions) {
	printedDefault := false
	for workingFile := range outputChannel {
		switch opts.Format {
		case models.FormatLong:
			List(w, workingFile, opts.ShowHidden, opts.NoColor, opts.Hash)
		case models.FormatCompare:
			Comparable(w, workingFile)
		case models.FormatJSON:
			JSON(w, workingFile)
		case models.FormatDefault, "":
			Default(w, workingFile, opts.ShowHidden, opts.NoColor)
			printedDefault = true
		default:
			Default(w, workingFile, opts.ShowHidden, opts.NoColor)
			printedDefault = true
		}
	}

	if printedDefault {
		fmt.Fprintln(w)
	}
}
