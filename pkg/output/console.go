package output

import (
	"fmt"
	"os"
)

func Default(files []os.FileInfo, showHidden bool) {
	for _, file := range files {
		if file.Name()[0] != '.' || showHidden {
			fmt.Printf(" %s ", file.Name())
		}
	}
	fmt.Printf("\n")
}
