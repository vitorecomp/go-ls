package looker

import (
	"io/ioutil"
	"log"
	"sync"

	"github.com/vitorecomp/go-ls/pkg/models"
)

func Look(wg *sync.WaitGroup, path string, parameters models.LookParameters, channel chan models.File) {
	defer wg.Done()
	look(path, parameters, channel)
}

func look(path string, parameters models.LookParameters, channel chan models.File) {
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
			look(fileModel.GetAbsolutePath(), parameters, channel)
		}
		channel <- fileModel
	}

}
