package looker

import (
	"io/ioutil"
	"log"

	"github.com/vitorecomp/go-ls/pkg/models"
)

func Look(path string, recursive bool) []models.File {
	filesSo, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	files := make([]models.File, len(filesSo))

	for i, file := range filesSo {
		files[i] = models.File{
			FileInfo: file,
		}
	}
	return files
}
