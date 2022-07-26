package looker

import (
	"crypto/sha256"
	"encoding/base64"
	"sync"

	"github.com/vitorecomp/go-ls/pkg/models"
)

var WORKERS = 15

func hashDir(name string, internalHashs []string) string {
	hash := name
	for _, dirhashs := range internalHashs {
		hash += dirhashs
	}
	byteHash := (sha256.Sum256([]byte(hash)))
	hash = base64.URLEncoding.EncodeToString(byteHash[:])
	return hash
}

func Hash(wg *sync.WaitGroup, hashChannel chan models.File, outputChannel chan models.File, parameters models.LookParameters) {
	var hashWg sync.WaitGroup
	for i := 0; i < WORKERS; i++ {
		go func() {
			hashWg.Add(1)
			for file := range hashChannel {
				if !file.FileInfo.IsDir() && parameters.Hash {
					file.Hash = file.GetHash()
				}
				outputChannel <- file
			}
			defer hashWg.Done()
		}()
	}
	hashWg.Wait()
	defer wg.Done()
}
