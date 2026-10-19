// Package store provides persistence and query operations (scans, diffs, duplicates)
// for the go-ls server.
package store

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/vitorecomp/go-ls/pkg/models"
)

// ErrScanNotFound is returned when a requested scan ID does not exist in the store.
var ErrScanNotFound = errors.New("scan not found")

// Store defines the persistence contract for storing, comparing, and analyzing directory scans.
type Store interface {
	SaveScan(ctx context.Context, scan models.ScanRecord) (*models.ScanSummary, error)
	SaveScanBatch(ctx context.Context, req models.ScanBatchRequest) (*models.ScanSummary, error)
	ListScans(ctx context.Context) ([]models.ScanSummary, error)
	ListComputers(ctx context.Context) ([]models.Computer, error)
	GetScan(ctx context.Context, scanID string) (*models.ScanRecord, error)
	GetScanHistory(ctx context.Context, scanID string) ([]models.ScanHistoryEntry, error)
	DiffScans(ctx context.Context, baseScanID, targetScanID string) (*models.DiffReport, error)
	FindDuplicates(ctx context.Context, scanID string) ([]models.DuplicateGroup, error)
	Close() error
}

// Open initializes a Store from a connection URL.
// If dbURL is empty or starts with "memory://", an in-memory store is returned.
// Otherwise, a PostgreSQL store is opened and migrated.
func Open(ctx context.Context, dbURL string) (Store, error) {
	trimmed := strings.TrimSpace(dbURL)
	if trimmed == "" || strings.HasPrefix(trimmed, "memory://") || trimmed == "memory" {
		return NewMemoryStore(), nil
	}
	return NewPostgresStore(ctx, trimmed)
}

// NormalizeScan prepares a ScanRecord before persistence by ensuring ID, ComputerID,
// DriveID, CreatedAt, UpdatedAt, Version, and summary counts are populated.
func NormalizeScan(scan models.ScanRecord) models.ScanRecord {
	if strings.TrimSpace(scan.ID) == "" {
		scan.ID = generateScanID()
	}
	if strings.TrimSpace(scan.Hostname) == "" {
		if strings.TrimSpace(scan.ComputerID) != "" {
			scan.Hostname = scan.ComputerID
		} else {
			scan.Hostname = "unknown-host"
		}
	}
	if strings.TrimSpace(scan.ComputerID) == "" {
		scan.ComputerID = scan.Hostname
	}
	if strings.TrimSpace(scan.DriveID) == "" {
		scan.DriveID = "default-drive"
	}
	now := time.Now().UTC()
	if scan.CreatedAt.IsZero() {
		scan.CreatedAt = now
	}
	if scan.UpdatedAt.IsZero() {
		scan.UpdatedAt = scan.CreatedAt
	}
	if scan.Version <= 0 {
		scan.Version = 1
	}
	if scan.HashAlgo == "" {
		scan.HashAlgo = string(models.HashCRC32)
	}
	if scan.Status == "" {
		scan.Status = models.ScanStatusCompleted
	}

	var files, dirs int
	var totalBytes int64
	for _, e := range scan.Entries {
		if e.IsDir {
			dirs++
		} else {
			files++
			totalBytes += e.Size
		}
	}
	scan.TotalFiles = files
	scan.TotalDirs = dirs
	scan.TotalBytes = totalBytes
	return scan
}

// NormalizeScanBatch populates default fields on a ScanBatchRequest.
func NormalizeScanBatch(req models.ScanBatchRequest) models.ScanBatchRequest {
	if strings.TrimSpace(req.ID) == "" {
		req.ID = generateScanID()
	}
	if strings.TrimSpace(req.Hostname) == "" {
		if strings.TrimSpace(req.ComputerID) != "" {
			req.Hostname = req.ComputerID
		} else {
			req.Hostname = "unknown-host"
		}
	}
	if strings.TrimSpace(req.ComputerID) == "" {
		req.ComputerID = req.Hostname
	}
	if strings.TrimSpace(req.DriveID) == "" {
		req.DriveID = "default-drive"
	}
	if req.HashAlgo == "" {
		req.HashAlgo = string(models.HashCRC32)
	}
	return req
}

// UpsertScanEntries merges incoming batch entries into existing entries by RelPath,
// returning the sorted slice and recalculated (totalFiles, totalDirs, totalBytes).
func UpsertScanEntries(existing []models.ScanEntry, incoming []models.ScanEntry) ([]models.ScanEntry, int, int, int64) {
	byPath := make(map[string]models.ScanEntry, len(existing)+len(incoming))
	for _, e := range existing {
		byPath[e.RelPath] = e
	}
	for _, e := range incoming {
		byPath[e.RelPath] = e
	}

	merged := make([]models.ScanEntry, 0, len(byPath))
	var files, dirs int
	var totalBytes int64
	for _, e := range byPath {
		merged = append(merged, e)
		if e.IsDir {
			dirs++
		} else {
			files++
			totalBytes += e.Size
		}
	}
	slices.SortFunc(merged, func(a, b models.ScanEntry) int {
		return cmp.Compare(a.RelPath, b.RelPath)
	})
	return merged, files, dirs, totalBytes
}

// MergeScanUpdate combines an incoming ScanRecord with an optional existing ScanRecord
// for the same (ComputerID, DriveID, RootPath) directory, updating the snapshot in place
// and recording the revision diff in History.
func MergeScanUpdate(existing *models.ScanRecord, incoming models.ScanRecord) models.ScanRecord {
	if existing != nil && strings.TrimSpace(incoming.ID) == "" {
		incoming.ID = existing.ID
	}
	normalized := NormalizeScan(incoming)

	if existing == nil {
		normalized.Version = 1
		initialDiff := ComputeDiff(models.ScanRecord{}, normalized)
		normalized.History = []models.ScanHistoryEntry{
			{
				Version:    1,
				Tag:        normalized.Tag,
				RootHash:   normalized.RootHash,
				TotalFiles: normalized.TotalFiles,
				TotalDirs:  normalized.TotalDirs,
				TotalBytes: normalized.TotalBytes,
				Identical:  false,
				Added:      initialDiff.Added,
				Deleted:    initialDiff.Deleted,
				Modified:   initialDiff.Modified,
				Changes:    initialDiff.Changes,
				CreatedAt:  normalized.CreatedAt,
			},
		}
		return normalized
	}

	// Preserve existing snapshot identity and original creation timestamp.
	normalized.ID = existing.ID
	normalized.CreatedAt = existing.CreatedAt
	if normalized.UpdatedAt.Before(existing.UpdatedAt) || normalized.UpdatedAt.Equal(existing.CreatedAt) {
		normalized.UpdatedAt = time.Now().UTC()
	}
	if normalized.Tag == "" && existing.Tag != "" {
		normalized.Tag = existing.Tag
	}

	diff := ComputeDiff(*existing, normalized)
	history := make([]models.ScanHistoryEntry, len(existing.History))
	copy(history, existing.History)

	if !diff.Identical || len(history) == 0 {
		normalized.Version = existing.Version + 1
		if len(history) == 0 && existing.Version <= 0 {
			normalized.Version = 1
		}
		newEntry := models.ScanHistoryEntry{
			Version:      normalized.Version,
			Tag:          normalized.Tag,
			PreviousHash: existing.RootHash,
			RootHash:     normalized.RootHash,
			TotalFiles:   normalized.TotalFiles,
			TotalDirs:    normalized.TotalDirs,
			TotalBytes:   normalized.TotalBytes,
			Identical:    diff.Identical,
			Added:        diff.Added,
			Deleted:      diff.Deleted,
			Modified:     diff.Modified,
			Changes:      diff.Changes,
			CreatedAt:    normalized.UpdatedAt,
		}
		history = append([]models.ScanHistoryEntry{newEntry}, history...)
	} else {
		normalized.Version = existing.Version
		if normalized.Version <= 0 {
			normalized.Version = 1
		}
	}

	sort.Slice(history, func(i, j int) bool {
		return history[i].Version > history[j].Version
	})
	normalized.History = history
	return normalized
}

// GroupScansByComputer organizes directory scan summaries under their parent Computer.
func GroupScansByComputer(summaries []models.ScanSummary) []models.Computer {
	byComp := make(map[string]*models.Computer)
	driveSet := make(map[string]map[string]struct{})

	for _, s := range summaries {
		compID := s.ComputerID
		if compID == "" {
			compID = s.Hostname
		}
		comp, ok := byComp[compID]
		if !ok {
			comp = &models.Computer{
				ID:       compID,
				Hostname: s.Hostname,
			}
			byComp[compID] = comp
			driveSet[compID] = make(map[string]struct{})
		}
		if s.Hostname != "" && (comp.Hostname == "" || comp.Hostname == compID) {
			comp.Hostname = s.Hostname
		}
		if s.DriveID != "" {
			driveSet[compID][s.DriveID] = struct{}{}
		}
		comp.Directories = append(comp.Directories, s)
		comp.TotalFiles += s.TotalFiles
		comp.TotalDirs += s.TotalDirs
		comp.TotalBytes += s.TotalBytes

		ts := s.UpdatedAt
		if ts.IsZero() {
			ts = s.CreatedAt
		}
		if ts.After(comp.UpdatedAt) {
			comp.UpdatedAt = ts
		}
	}

	result := make([]models.Computer, 0, len(byComp))
	for compID, comp := range byComp {
		drives := make([]string, 0, len(driveSet[compID]))
		for d := range driveSet[compID] {
			drives = append(drives, d)
		}
		sort.Strings(drives)
		comp.Drives = drives

		sort.Slice(comp.Directories, func(i, j int) bool {
			return comp.Directories[i].RootPath < comp.Directories[j].RootPath
		})
		result = append(result, *comp)
	}

	sort.Slice(result, func(i, j int) bool {
		if !result[i].UpdatedAt.Equal(result[j].UpdatedAt) {
			return result[i].UpdatedAt.After(result[j].UpdatedAt)
		}
		return result[i].Hostname < result[j].Hostname
	})
	return result
}

func generateScanID() string {
	return models.GenerateScanID()
}

// ComputeDiff compares two ScanRecords by relative path and reports added, deleted,
// and modified files or directories.
func ComputeDiff(base, target models.ScanRecord) *models.DiffReport {
	return models.ComputeDiff(base, target)
}

// ComputeDuplicates identifies non-empty regular files that share the same content hash and size.
func ComputeDuplicates(scans []models.ScanRecord) []models.DuplicateGroup {
	type key struct {
		hash string
		size int64
	}

	groups := make(map[key][]models.DuplicateFileRef)
	for _, scan := range scans {
		for _, entry := range scan.Entries {
			if entry.IsDir || entry.Hash == "" || entry.Size == 0 {
				continue
			}
			k := key{hash: entry.Hash, size: entry.Size}
			groups[k] = append(groups[k], models.DuplicateFileRef{
				ScanID:     scan.ID,
				ComputerID: scan.ComputerID,
				Hostname:   scan.Hostname,
				DriveID:    scan.DriveID,
				RootPath:   scan.RootPath,
				RelPath:    entry.RelPath,
				AbsPath:    entry.AbsPath,
			})
		}
	}

	var result []models.DuplicateGroup
	for k, refs := range groups {
		if len(refs) < 2 {
			continue
		}
		sort.Slice(refs, func(i, j int) bool {
			if refs[i].ScanID != refs[j].ScanID {
				return refs[i].ScanID < refs[j].ScanID
			}
			return refs[i].RelPath < refs[j].RelPath
		})
		result = append(result, models.DuplicateGroup{
			Hash:        k.hash,
			Size:        k.size,
			Count:       len(refs),
			WastedBytes: int64(len(refs)-1) * k.size,
			Files:       refs,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].WastedBytes != result[j].WastedBytes {
			return result[i].WastedBytes > result[j].WastedBytes
		}
		return result[i].Hash < result[j].Hash
	})

	return result
}
