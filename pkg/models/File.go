package models

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"syscall"
	"time"
)

type File struct {
	Dir      bool
	FileInfo os.FileInfo
	Owner    user.User
	Group    user.Group
	Hash     string
	BasePath string

	CreatedAt string
	UpdatedAt string
	AccessAt  string
}

func (f *File) GetTimeStamps() {
	stat := f.FileInfo.Sys().(*syscall.Stat_t)
	atime := time.Unix(stat.Atim.Sec, stat.Atim.Nsec)
	ctime := time.Unix(stat.Ctim.Sec, stat.Ctim.Nsec)
	f.AccessAt = atime.String()
	f.UpdatedAt = f.FileInfo.ModTime().String()
	f.CreatedAt = ctime.String()
}

// calculate the cr32 of a file
func (f *File) GetChecksum() {
	if f.Dir {
		panic("Cannot calculate checksum of a directory")
	}
	fileAbsoluteName := f.GetAbsolutePath()
	file, err := os.Open(fileAbsoluteName)
	if err != nil {
		// Non-fatal error for individual files
		log.Printf("Error reading file %s: %v", fileAbsoluteName, err)
		f.Hash = ""
		return
	}
	defer file.Close()

	hasher := crc32.NewIEEE()
	if _, err := io.Copy(hasher, file); err != nil {
		log.Printf("Error calculating checksum for %s: %v", fileAbsoluteName, err)
	}
	f.Hash = fmt.Sprintf("%x", hasher.Sum32())
}

func (f *File) GetSize() {
	f.FileInfo.Size()
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
