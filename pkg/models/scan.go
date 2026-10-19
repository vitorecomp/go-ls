package models

import (
	"cmp"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// ScanStatusInProgress indicates a scan is currently receiving incremental batches.
	ScanStatusInProgress = "in_progress"
	// ScanStatusCompleted indicates all batches have been received and the snapshot is finalized.
	ScanStatusCompleted = "completed"
)

// GenerateScanID returns a unique time-prefixed scan identifier.
func GenerateScanID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("scan-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("scan-%s-%s", time.Now().UTC().Format("20060102-150405"), hex.EncodeToString(b))
}

// ScanEntry represents a single file or directory recorded during a client scan.
type ScanEntry struct {
	RelPath string    `json:"rel_path"`
	AbsPath string    `json:"abs_path"`
	Name    string    `json:"name"`
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	Mode    string    `json:"mode"`
	Owner   string    `json:"owner"`
	Group   string    `json:"group"`
	ModTime time.Time `json:"mod_time"`
	Hash    string    `json:"hash"`
}

// ScanSummary holds high-level metadata about a stored directory snapshot inside a computer.
type ScanSummary struct {
	ID         string    `json:"id"`
	Tag        string    `json:"tag,omitempty"`
	ComputerID string    `json:"computer_id"`
	Hostname   string    `json:"hostname"`
	DriveID    string    `json:"drive_id"`
	RootPath   string    `json:"root_path"`
	HashAlgo   string    `json:"hash_algo"`
	RootHash   string    `json:"root_hash"`
	Status     string    `json:"status,omitempty"`
	TotalFiles int       `json:"total_files"`
	TotalDirs  int       `json:"total_dirs"`
	TotalBytes int64     `json:"total_bytes"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ScanBatchRequest represents an incremental batch of scan entries sent to the server.
type ScanBatchRequest struct {
	ID         string      `json:"id"`
	Tag        string      `json:"tag,omitempty"`
	ComputerID string      `json:"computer_id"`
	Hostname   string      `json:"hostname"`
	DriveID    string      `json:"drive_id"`
	RootPath   string      `json:"root_path"`
	HashAlgo   string      `json:"hash_algo"`
	RootHash   string      `json:"root_hash,omitempty"`
	Entries    []ScanEntry `json:"entries,omitempty"`
	BatchIndex int         `json:"batch_index"`
	Final      bool        `json:"final"`
	Resume     bool        `json:"resume,omitempty"`
}

// ScanHistoryEntry records a single revision and change diff in the history of a tracked directory.
type ScanHistoryEntry struct {
	Version      int        `json:"version"`
	Tag          string     `json:"tag,omitempty"`
	PreviousHash string     `json:"previous_hash,omitempty"`
	RootHash     string     `json:"root_hash"`
	TotalFiles   int        `json:"total_files"`
	TotalDirs    int        `json:"total_dirs"`
	TotalBytes   int64      `json:"total_bytes"`
	Identical    bool       `json:"identical"`
	Added        int        `json:"added"`
	Deleted      int        `json:"deleted"`
	Modified     int        `json:"modified"`
	Changes      []DiffItem `json:"changes,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ScanRecord contains the directory snapshot summary, its current filesystem entries,
// and its historical revisions.
type ScanRecord struct {
	ScanSummary
	Entries []ScanEntry        `json:"entries"`
	History []ScanHistoryEntry `json:"history,omitempty"`
}

// Computer represents a machine that contains one or more tracked directory snapshots across its drives.
type Computer struct {
	ID          string        `json:"id"`
	Hostname    string        `json:"hostname"`
	Drives      []string      `json:"drives"`
	Directories []ScanSummary `json:"directories"`
	TotalFiles  int           `json:"total_files"`
	TotalDirs   int           `json:"total_dirs"`
	TotalBytes  int64         `json:"total_bytes"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// DiffChangeType describes how an entry changed between two scans.
type DiffChangeType string

const (
	// DiffAdded indicates an entry exists in the target scan but not in the base scan.
	DiffAdded DiffChangeType = "added"
	// DiffDeleted indicates an entry existed in the base scan but is missing from the target scan.
	DiffDeleted DiffChangeType = "deleted"
	// DiffModified indicates an entry exists in both scans with different content hashes, sizes, or types.
	DiffModified DiffChangeType = "modified"
)

// DiffItem describes a single difference between a base scan and a target scan.
type DiffItem struct {
	RelPath    string         `json:"rel_path"`
	Change     DiffChangeType `json:"change"`
	IsDir      bool           `json:"is_dir"`
	BaseHash   string         `json:"base_hash,omitempty"`
	TargetHash string         `json:"target_hash,omitempty"`
	BaseSize   int64          `json:"base_size,omitempty"`
	TargetSize int64          `json:"target_size,omitempty"`
}

// DiffReport summarizes all differences between two scans stored on the server.
type DiffReport struct {
	BaseScan   ScanSummary `json:"base_scan"`
	TargetScan ScanSummary `json:"target_scan"`
	Identical  bool        `json:"identical"`
	Added      int         `json:"added"`
	Deleted    int         `json:"deleted"`
	Modified   int         `json:"modified"`
	Unchanged  int         `json:"unchanged"`
	Changes    []DiffItem  `json:"changes"`
}

// LiveEvent represents a batch of filesystem changes detected during live directory monitoring.
type LiveEvent struct {
	Timestamp   time.Time  `json:"timestamp"`
	RootPath    string     `json:"root_path"`
	RootHash    string     `json:"root_hash"`
	Added       int        `json:"added"`
	Deleted     int        `json:"deleted"`
	Modified    int        `json:"modified"`
	Changes     []DiffItem `json:"changes"`
	ScanID      string     `json:"scan_id,omitempty"`
	ScanVersion int        `json:"scan_version,omitempty"`
}

// DuplicateFileRef identifies a specific file instance within a stored scan.
type DuplicateFileRef struct {
	ScanID     string `json:"scan_id"`
	ComputerID string `json:"computer_id,omitempty"`
	Hostname   string `json:"hostname"`
	DriveID    string `json:"drive_id,omitempty"`
	RootPath   string `json:"root_path"`
	RelPath    string `json:"rel_path"`
	AbsPath    string `json:"abs_path"`
}

// DuplicateGroup groups files that share the same non-empty content hash and size.
type DuplicateGroup struct {
	Hash        string             `json:"hash"`
	Size        int64              `json:"size"`
	Count       int                `json:"count"`
	WastedBytes int64              `json:"wasted_bytes"`
	Files       []DuplicateFileRef `json:"files"`
}

// FileToScanEntry converts a discovered File into a portable ScanEntry with a path
// relative to rootPath.
func FileToScanEntry(rootPath string, f File) ScanEntry {
	absRoot := rootPath
	if !filepath.IsAbs(absRoot) {
		if resolved, err := filepath.Abs(rootPath); err == nil {
			absRoot = resolved
		} else {
			absRoot = filepath.Clean(rootPath)
		}
	}
	return fileToScanEntryCleanRoot(absRoot, f)
}

func fileToScanEntryCleanRoot(absRoot string, f File) ScanEntry {
	absPath := f.GetAbsolutePath()
	relPath, err := filepath.Rel(absRoot, absPath)
	if err != nil || relPath == "" || relPath == "." {
		if f.FileInfo != nil {
			relPath = f.FileInfo.Name()
		}
	}
	relPath = filepath.ToSlash(relPath)

	var (
		name    string
		size    int64
		mode    string
		modTime time.Time
	)
	if f.FileInfo != nil {
		name = f.FileInfo.Name()
		size = f.FileInfo.Size()
		mode = f.FileInfo.Mode().String()
		modTime = f.FileInfo.ModTime().UTC()
	}

	return ScanEntry{
		RelPath: relPath,
		AbsPath: absPath,
		Name:    name,
		IsDir:   f.Dir,
		Size:    size,
		Mode:    mode,
		Owner:   f.Owner.Username,
		Group:   f.Group.Name,
		ModTime: modTime,
		Hash:    f.Hash,
	}
}

// BuildScanRecord assembles a ScanRecord from a slice of discovered File objects,
// sorting entries deterministically by RelPath and computing the root Merkle hash.
func BuildScanRecord(id, tag, hostname, rootPath string, algo HashAlgorithm, files []File) ScanRecord {
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		absRoot = filepath.Clean(rootPath)
	}
	if algo == "" {
		algo = HashCRC32
	}

	n := len(files)
	entries := make([]ScanEntry, n)
	var totalFiles, totalDirs int
	var totalBytes int64

	numCPU := runtime.GOMAXPROCS(0)
	if n >= 512 && numCPU > 1 {
		var atomicFiles, atomicDirs, atomicBytes atomic.Int64
		var wg sync.WaitGroup
		chunkSize := (n + numCPU - 1) / numCPU
		for w := 0; w < numCPU; w++ {
			start := w * chunkSize
			if start >= n {
				break
			}
			end := start + chunkSize
			if end > n {
				end = n
			}
			wg.Add(1)
			go func(lo, hi int) {
				defer wg.Done()
				var localFiles, localDirs, localBytes int64
				for idx := lo; idx < hi; idx++ {
					entry := fileToScanEntryCleanRoot(absRoot, files[idx])
					entries[idx] = entry
					if entry.IsDir {
						localDirs++
					} else {
						localFiles++
						localBytes += entry.Size
					}
				}
				atomicFiles.Add(localFiles)
				atomicDirs.Add(localDirs)
				atomicBytes.Add(localBytes)
			}(start, end)
		}
		wg.Wait()
		totalFiles = int(atomicFiles.Load())
		totalDirs = int(atomicDirs.Load())
		totalBytes = atomicBytes.Load()
	} else {
		for idx, f := range files {
			entry := fileToScanEntryCleanRoot(absRoot, f)
			entries[idx] = entry
			if entry.IsDir {
				totalDirs++
			} else {
				totalFiles++
				totalBytes += entry.Size
			}
		}
	}

	slices.SortFunc(entries, func(a, b ScanEntry) int {
		return cmp.Compare(a.RelPath, b.RelPath)
	})

	rootHash := ComputeRootHash(algo, entries)
	now := time.Now().UTC()

	return ScanRecord{
		ScanSummary: ScanSummary{
			ID:         id,
			Tag:        tag,
			ComputerID: DetectComputerID(hostname),
			Hostname:   hostname,
			DriveID:    DetectDriveID(absRoot),
			RootPath:   absRoot,
			HashAlgo:   string(algo),
			RootHash:   rootHash,
			Status:     ScanStatusCompleted,
			TotalFiles: totalFiles,
			TotalDirs:  totalDirs,
			TotalBytes: totalBytes,
			Version:    1,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		Entries: entries,
	}
}

// ComputeRootHash computes the Merkle root hash from direct children of rootPath in entries.
func ComputeRootHash(algo HashAlgorithm, entries []ScanEntry) string {
	if algo == "" {
		algo = HashCRC32
	}
	childHashes := make([]string, 0, len(entries))
	for _, e := range entries {
		if filepath.Dir(filepath.FromSlash(e.RelPath)) == "." {
			childHashes = append(childHashes, e.Hash)
		}
	}
	var rootHolder File
	return rootHolder.ComputeDirHash(algo, childHashes)
}

// ComputeDiff compares two ScanRecords by relative path and reports added, deleted,
// and modified files or directories.
func ComputeDiff(base, target ScanRecord) *DiffReport {
	baseMap := make(map[string]ScanEntry, len(base.Entries))
	for _, e := range base.Entries {
		baseMap[e.RelPath] = e
	}

	targetMap := make(map[string]ScanEntry, len(target.Entries))
	for _, e := range target.Entries {
		targetMap[e.RelPath] = e
	}

	var changes []DiffItem
	var added, deleted, modified, unchanged int

	for relPath, baseEntry := range baseMap {
		targetEntry, exists := targetMap[relPath]
		if !exists {
			deleted++
			changes = append(changes, DiffItem{
				RelPath:  relPath,
				Change:   DiffDeleted,
				IsDir:    baseEntry.IsDir,
				BaseHash: baseEntry.Hash,
				BaseSize: baseEntry.Size,
			})
			continue
		}

		if baseEntry.IsDir != targetEntry.IsDir ||
			baseEntry.Hash != targetEntry.Hash ||
			(!baseEntry.IsDir && baseEntry.Size != targetEntry.Size) {
			modified++
			changes = append(changes, DiffItem{
				RelPath:    relPath,
				Change:     DiffModified,
				IsDir:      targetEntry.IsDir,
				BaseHash:   baseEntry.Hash,
				TargetHash: targetEntry.Hash,
				BaseSize:   baseEntry.Size,
				TargetSize: targetEntry.Size,
			})
		} else {
			unchanged++
		}
	}

	for relPath, targetEntry := range targetMap {
		if _, exists := baseMap[relPath]; !exists {
			added++
			changes = append(changes, DiffItem{
				RelPath:    relPath,
				Change:     DiffAdded,
				IsDir:      targetEntry.IsDir,
				TargetHash: targetEntry.Hash,
				TargetSize: targetEntry.Size,
			})
		}
	}

	slices.SortFunc(changes, func(a, b DiffItem) int {
		return cmp.Compare(a.RelPath, b.RelPath)
	})

	return &DiffReport{
		BaseScan:   base.ScanSummary,
		TargetScan: target.ScanSummary,
		Identical:  len(changes) == 0,
		Added:      added,
		Deleted:    deleted,
		Modified:   modified,
		Unchanged:  unchanged,
		Changes:    changes,
	}
}
