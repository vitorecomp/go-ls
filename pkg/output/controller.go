package output

import (
	"fmt"

	"github.com/vitorecomp/go-ls/pkg/models"
)

func Output(outputChannel chan models.File, outputParameters models.OutputParameters) {
	for workingFile := range outputChannel {
		fmt.Println(workingFile.GetAbsolutePath())
	}
}
