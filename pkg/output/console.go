// Package output formats and streams discovered File entries to the console or an io.Writer.
package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/vitorecomp/go-ls/pkg/cli"
	"github.com/vitorecomp/go-ls/pkg/models"
)

// Default renders a single entry in compact inline format.
func Default(w io.Writer, file models.File, showHidden bool, noColor bool) {
	if file.FileInfo == nil {
		return
	}
	name := file.FileInfo.Name()
	if strings.HasPrefix(name, ".") && !showHidden {
		return
	}

	terminal := newColorPrinter(noColor, file.FileInfo.IsDir())
	terminal.Fprintf(w, "%s  ", name)
}

// List renders a single entry in long (-l) format including mode, owner, group, size,
// modification timestamp, optional checksum, and entry name.
func List(w io.Writer, file models.File, showHidden bool, noColor bool, showHash bool) {
	if file.FileInfo == nil {
		return
	}
	name := file.FileInfo.Name()
	if strings.HasPrefix(name, ".") && !showHidden {
		return
	}

	file.FillOwnerAndGroup()

	fmt.Fprintf(w, "%-10s %-8s %-8s %8d %s",
		file.FileInfo.Mode(),
		file.Owner.Username,
		file.Group.Name,
		file.FileInfo.Size(),
		file.FileInfo.ModTime().Format(cli.Constants.DateFormat),
	)

	if showHash {
		hashVal := file.Hash
		if hashVal == "" {
			hashVal = "-"
		}
		fmt.Fprintf(w, " %-8s", hashVal)
	}

	fmt.Fprint(w, " ")
	namePrinter := newColorPrinter(noColor, file.FileInfo.IsDir())
	namePrinter.Fprintln(w, name)
}

// Comparable renders a single entry as an RFC 4180 CSV record: hash,size,type,path.
// Paths containing commas, quotes, or spaces are automatically escaped.
func Comparable(w io.Writer, file models.File) {
	if file.FileInfo == nil {
		return
	}
	entryType := "file"
	if file.FileInfo.IsDir() {
		entryType = "directory"
	}

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		file.Hash,
		strconv.FormatInt(file.FileInfo.Size(), 10),
		entryType,
		file.GetAbsolutePath(),
	})
	cw.Flush()
}

// JSON renders a single entry as a newline-delimited JSON object.
func JSON(w io.Writer, file models.File) {
	enc := json.NewEncoder(w)
	_ = enc.Encode(file)
}

func newColorPrinter(noColor bool, isDir bool) *color.Color {
	var c *color.Color
	if isDir {
		c = color.New(color.Bold, color.FgBlue)
	} else {
		c = color.New()
	}
	if noColor {
		c.DisableColor()
	} else {
		c.EnableColor()
	}
	return c
}
