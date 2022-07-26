package models

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"syscall"
)

type File struct {
	FileInfo os.FileInfo
	Owner    user.User
	Group    user.Group
	Hash     string
	BasePath string
}

func (f *File) FillOwnerAndGroup() {
	stat_t := f.FileInfo.Sys().(*syscall.Stat_t)

	uid := fmt.Sprint(stat_t.Uid)
	gid := fmt.Sprint(stat_t.Gid)

	fileOwner, err := user.LookupId(uid)
	if err != nil {
		f.Owner = user.User{
			Username: uid,
		}
	} else {
		f.Owner = *fileOwner

	}

	group, err := user.LookupGroupId(gid)
	if err != nil {
		f.Group = user.Group{
			Name: gid,
		}
	} else {
		f.Group = *group
	}
}

func (f *File) GetAbsolutePath() string {
	slash := ""
	if f.BasePath[len(f.BasePath)-1] != '/' {
		slash += "/"
	}
	path, err := filepath.Abs(f.BasePath + slash + f.FileInfo.Name())
	if err != nil {
		log.Fatal(err)
	}
	return path
}

func (f *File) GetHash() string {
	fileAbsoluteName := f.GetAbsolutePath()
	file, err := os.Open(fileAbsoluteName)
	if err != nil {
		errl := log.New(os.Stderr, "", 0)
		errl.Println("Error reading file " + fileAbsoluteName)
		errl.Println(err)
		return ""
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
