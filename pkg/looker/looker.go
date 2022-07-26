package looker

import (
	"io/ioutil"
	"log"

	"github.com/vitorecomp/go-ls/pkg/models"
)

func Look(path string, parameters models.LookParameters, hashChannel chan models.File) {
	filesSo, err := ioutil.ReadDir(path)

	if err != nil {
		log.Fatal(err)
	}

	for _, file := range filesSo {
		fileModel := models.File{
			FileInfo: file,
			BasePath: path,
		}
		if file.IsDir() {
			Look(fileModel.GetAbsolutePath(), parameters, hashChannel)
		}
		hashChannel <- fileModel
	}

}
