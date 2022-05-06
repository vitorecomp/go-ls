package looker

import (
	"io/ioutil"
	"log"
	"os"
)

func Look(path string, recursive bool) []os.FileInfo {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}
	return files
}
