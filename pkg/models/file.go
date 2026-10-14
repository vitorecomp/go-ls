package models

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

var (
	uidCache sync.Map // map[string]user.User
	gidCache sync.Map // map[string]user.Group

	// ErrDirectoryChecksum is returned when attempting to compute a file checksum on a directory.
	ErrDirectoryChecksum = errors.New("cannot calculate checksum of a directory")
)

// File holds metadata and optional content checksums for a single filesystem entry.
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

// FileJSON represents the serialized JSON structure for a File when using --json output.
type FileJSON struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Type      string `json:"type"`
	IsDir     bool   `json:"is_dir"`
	Size      int64  `json:"size"`
	Mode      string `json:"mode"`
	Owner     string `json:"owner"`
	Group     string `json:"group"`
	ModTime   string `json:"mod_time"`
	AccessAt  string `json:"access_time,omitempty"`
	CreatedAt string `json:"change_time,omitempty"`
	Hash      string `json:"hash,omitempty"`
}

// NewFile constructs a File from a base directory path and os.FileInfo,
// populating owner/group and timestamp metadata.
func NewFile(basePath string, info os.FileInfo) File {
	f := File{
		Dir:      info.IsDir(),
		FileInfo: info,
		BasePath: basePath,
	}
	f.FillOwnerAndGroup()
	f.GetTimeStamps()
	return f
}

// MarshalJSON implements json.Marshaler so File serializes cleanly in --json mode.
func (f File) MarshalJSON() ([]byte, error) {
	var (
		name    string
		size    int64
		mode    string
		modTime string
	)
	if f.FileInfo != nil {
		name = f.FileInfo.Name()
		size = f.FileInfo.Size()
		mode = f.FileInfo.Mode().String()
		modTime = f.FileInfo.ModTime().Format(time.RFC3339)
	}
	entryType := "file"
	if f.Dir {
		entryType = "directory"
	}
	rec := FileJSON{
		Name:      name,
		Path:      f.GetAbsolutePath(),
		Type:      entryType,
		IsDir:     f.Dir,
		Size:      size,
		Mode:      mode,
		Owner:     f.Owner.Username,
		Group:     f.Group.Name,
		ModTime:   modTime,
		AccessAt:  f.AccessAt,
		CreatedAt: f.CreatedAt,
		Hash:      f.Hash,
	}
	return json.Marshal(rec)
}

// GetTimeStamps populates AccessAt, UpdatedAt, and CreatedAt from the underlying os.FileInfo.
func (f *File) GetTimeStamps() {
	if f.FileInfo == nil {
		return
	}
	modTime := f.FileInfo.ModTime()
	f.UpdatedAt = modTime.Format(time.RFC3339)

	stat, ok := f.FileInfo.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		f.AccessAt = f.UpdatedAt
		f.CreatedAt = f.UpdatedAt
		return
	}

	atime := time.Unix(stat.Atim.Sec, stat.Atim.Nsec)
	ctime := time.Unix(stat.Ctim.Sec, stat.Ctim.Nsec)
	f.AccessAt = atime.Format(time.RFC3339)
	f.CreatedAt = ctime.Format(time.RFC3339)
}

// FillOwnerAndGroup resolves the Unix UID and GID of the file into Owner and Group,
// caching lookups in memory to avoid repeated /etc/passwd and /etc/group reads.
func (f *File) FillOwnerAndGroup() {
	if f.FileInfo == nil {
		return
	}
	// Avoid redundant lookup if already populated.
	if f.Owner.Username != "" && f.Group.Name != "" {
		return
	}

	stat, ok := f.FileInfo.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return
	}

	uid := fmt.Sprint(stat.Uid)
	gid := fmt.Sprint(stat.Gid)

	if cachedUser, found := uidCache.Load(uid); found {
		f.Owner = cachedUser.(user.User)
	} else {
		if fileOwner, err := user.LookupId(uid); err == nil && fileOwner != nil {
			f.Owner = *fileOwner
		} else {
			f.Owner = user.User{Uid: uid, Username: uid}
		}
		uidCache.Store(uid, f.Owner)
	}

	if cachedGroup, found := gidCache.Load(gid); found {
		f.Group = cachedGroup.(user.Group)
	} else {
		if group, err := user.LookupGroupId(gid); err == nil && group != nil {
			f.Group = *group
		} else {
			f.Group = user.Group{Gid: gid, Name: gid}
		}
		gidCache.Store(gid, f.Group)
	}
}

// GetAbsolutePath returns the cleaned absolute path to the file.
func (f *File) GetAbsolutePath() string {
	var joined string
	switch {
	case f.FileInfo == nil:
		joined = f.BasePath
	case f.BasePath == "":
		joined = f.FileInfo.Name()
	default:
		joined = filepath.Join(f.BasePath, f.FileInfo.Name())
	}

	absPath, err := filepath.Abs(joined)
	if err != nil {
		return filepath.Clean(joined)
	}
	return absPath
}

// ComputeHash calculates the file content digest using the requested HashAlgorithm
// (defaults to HashCRC32) and stores the hexadecimal string in f.Hash.
func (f *File) ComputeHash(algo HashAlgorithm) error {
	if f.Dir || (f.FileInfo != nil && f.FileInfo.IsDir()) {
		return ErrDirectoryChecksum
	}

	fileAbsoluteName := f.GetAbsolutePath()
	file, err := os.Open(fileAbsoluteName)
	if err != nil {
		f.Hash = ""
		return fmt.Errorf("opening file %s: %w", fileAbsoluteName, err)
	}
	defer file.Close()

	var hasher hash.Hash
	switch algo {
	case HashSHA256:
		hasher = sha256.New()
	case HashCRC32, "":
		hasher = crc32.NewIEEE()
	default:
		hasher = crc32.NewIEEE()
	}

	if _, err := io.Copy(hasher, file); err != nil {
		f.Hash = ""
		return fmt.Errorf("hashing file %s: %w", fileAbsoluteName, err)
	}

	f.Hash = fmt.Sprintf("%x", hasher.Sum(nil))
	return nil
}

// ComputeDirHash calculates a directory's hash by combining the hashes of its child entries
// using the requested HashAlgorithm (defaults to HashCRC32) and stores the hex digest in f.Hash.
func (f *File) ComputeDirHash(algo HashAlgorithm, childHashes []string) string {
	var hasher hash.Hash
	switch algo {
	case HashSHA256:
		hasher = sha256.New()
	case HashCRC32, "":
		hasher = crc32.NewIEEE()
	default:
		hasher = crc32.NewIEEE()
	}

	for _, h := range childHashes {
		_, _ = io.WriteString(hasher, h)
	}

	f.Hash = fmt.Sprintf("%x", hasher.Sum(nil))
	return f.Hash
}

// GetChecksum calculates the CRC32 checksum of the file and stores it in f.Hash.
func (f *File) GetChecksum() error {
	return f.ComputeHash(HashCRC32)
}

// GetHash calculates the SHA-256 hash of the file, stores it in f.Hash, and returns it.
func (f *File) GetHash() string {
	if err := f.ComputeHash(HashSHA256); err != nil {
		return ""
	}
	return f.Hash
}
