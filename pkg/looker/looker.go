package looker

import (
	"log"
	"os"
	"sync"

	"github.com/vitorecomp/go-ls/pkg/models"
)

func Look(wg *sync.WaitGroup, path string, parameters models.LookParameters, channel chan models.File) {
	defer wg.Done()
	look(wg, path, parameters, channel)
}

func look(wg *sync.WaitGroup, path string, parameters models.LookParameters, channel chan models.File) {

	filesSo, err := os.ReadDir(path)

	if err != nil {
		log.Output(2, err.Error())
		return
	}

	for _, files_or_dirs := range filesSo {
		if !parameters.ShowHidden && files_or_dirs.Name()[0] == '.' {
			continue
		}

		workingFile := models.File{
			Dir:      files_or_dirs.IsDir(),
			BasePath: path,
		}

		workingFile.FileInfo, err = files_or_dirs.Info()
		if err != nil {
			log.Fatal(err)
		}
		workingFile.FillOwnerAndGroup()
		workingFile.GetTimeStamps()

		if workingFile.Dir && parameters.Recursive {
			wg.Add(1)
			go Look(wg, workingFile.GetAbsolutePath(), parameters, channel)
		}
		if !workingFile.Dir {
			workingFile.GetSize()
			if parameters.Hash {
				workingFile.GetChecksum()
			}
		}

		channel <- workingFile
	}

}
