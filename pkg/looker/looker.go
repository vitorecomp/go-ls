package looker

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/vitorecomp/go-ls/pkg/models"
)

func Look(path string, parameters models.LookParameters, outputChannel chan models.File) []string {
	filesSo, err := ioutil.ReadDir(path)

	hashs := make([]string, 0)

	if err != nil {
		log.Fatal(err)
	}

	hash := ""
	for _, file := range filesSo {
		fileModel := models.File{
			FileInfo: file,
			Hash:     hash,
			BasePath: path,
		}
		if file.IsDir() {
			internalHashs := make([]string, 0)
			if parameters.Recursive {
				internalHashs = Look(fileModel.GetAbsolutePath(), parameters, outputChannel)
			}
			if parameters.Hash {
				fileModel.Hash = hashDir(file.Name(), internalHashs)
			}
		} else {
			if parameters.Hash {
				fileModel.Hash = hashFile(fileModel.GetAbsolutePath())
			}
		}
		outputChannel <- fileModel
		hashs = append(hashs, (file.Name() + fileModel.Hash))
	}

	return hashs
}
func hashDir(name string, internalHashs []string) string {
	hash := name
	for _, dirhashs := range internalHashs {
		hash += dirhashs
	}
	byteHash := (sha256.Sum256([]byte(hash)))
	hash = base64.URLEncoding.EncodeToString(byteHash[:])
	return hash
}

func hashFile(fileAbsoluteName string) string {
	file, err := os.Open(fileAbsoluteName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	buf := make([]byte, 30*1024)
	sha256 := sha256.New()
	for {
		n, err := file.Read(buf)
		if n > 0 {
			_, err := sha256.Write(buf[:n])
			if err != nil {
				log.Fatal(err)
			}
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatal(err)
		}
	}

	return base64.URLEncoding.EncodeToString(sha256.Sum(nil))
}
