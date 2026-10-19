package output

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/vitorecomp/go-ls/pkg/models"
)

const outputBufferSize = 64 * 1024

// Output consumes discovered files from outputChannel and prints them to os.Stdout
// according to outputParameters.
func Output(outputChannel <-chan models.File, outputParameters models.OutputParameters) {
	WriteOutput(os.Stdout, outputChannel, outputParameters)
}

// WriteOutput consumes discovered files from outputChannel and writes formatted output to w.
func WriteOutput(w io.Writer, outputChannel <-chan models.File, opts models.OutputOptions) {
	bw := bufio.NewWriterSize(w, outputBufferSize)
	defer func() {
		_ = bw.Flush()
	}()

	printedDefault := false
	count := 0
	for workingFile := range outputChannel {
		switch opts.Format {
		case models.FormatLong:
			List(bw, workingFile, opts.ShowHidden, opts.NoColor, opts.Hash)
		case models.FormatCompare:
			Comparable(bw, workingFile)
		case models.FormatJSON:
			JSON(bw, workingFile)
		case models.FormatDefault, "":
			Default(bw, workingFile, opts.ShowHidden, opts.NoColor)
			printedDefault = true
		default:
			Default(bw, workingFile, opts.ShowHidden, opts.NoColor)
			printedDefault = true
		}
		count++
		if count&0x3f == 0 {
			_ = bw.Flush()
		}
	}

	if printedDefault {
		fmt.Fprintln(bw)
	}
}
