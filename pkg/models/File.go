package models

import (
	"fmt"
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
