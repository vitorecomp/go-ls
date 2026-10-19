package store

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/vitorecomp/go-ls/pkg/models"
)

type memoryBaseline struct {
	rootHash string
	version  int
	entries  []models.ScanEntry
	history  []models.ScanHistoryEntry
}

// MemoryStore is a thread-safe in-memory implementation of Store.
type MemoryStore struct {
	mu        sync.RWMutex
	scans     map[string]models.ScanRecord
	baselines map[string]memoryBaseline
}

// NewMemoryStore creates an initialized MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		scans:     make(map[string]models.ScanRecord),
		baselines: make(map[string]memoryBaseline),
	}
}

// SaveScan stores or updates a directory ScanRecord in memory under its computer and drive,
// recording the change history when updating an existing directory snapshot.
func (m *MemoryStore) SaveScan(_ context.Context, scan models.ScanRecord) (*models.ScanSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	probe := NormalizeScan(scan)
	var existing *models.ScanRecord
	if scan.ID != "" {
		if rec, ok := m.scans[scan.ID]; ok {
			r := rec
			existing = &r
		}
	}
	if existing == nil {
		for _, rec := range m.scans {
			if rec.ComputerID == probe.ComputerID &&
				rec.DriveID == probe.DriveID &&
				rec.RootPath == probe.RootPath {
				r := rec
				existing = &r
				break
			}
		}
	}

	merged := MergeScanUpdate(existing, scan)
	merged.Status = models.ScanStatusCompleted
	delete(m.baselines, merged.ID)

	merged.Entries = cloneEntries(merged.Entries)
	merged.History = cloneHistory(merged.History)

	m.scans[merged.ID] = merged

	summary := merged.ScanSummary
	return &summary, nil
}

// SaveScanBatch appends or upserts a batch of scan entries for an in-progress scan,
// finalizing the root hash and history revision when req.Final is true.
func (m *MemoryStore) SaveScanBatch(_ context.Context, req models.ScanBatchRequest) (*models.ScanSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	norm := NormalizeScanBatch(req)
	var existing *models.ScanRecord
	if req.ID != "" {
		if rec, ok := m.scans[req.ID]; ok {
			r := rec
			existing = &r
		}
	}
	if existing == nil && norm.RootPath != "" {
		for _, rec := range m.scans {
			if rec.ComputerID == norm.ComputerID &&
				rec.DriveID == norm.DriveID &&
				rec.RootPath == norm.RootPath {
				r := rec
				existing = &r
				break
			}
		}
	}

	if req.Resume && existing == nil && norm.RootPath == "" {
		return nil, fmt.Errorf("%w: %s", ErrScanNotFound, req.ID)
	}

	now := time.Now().UTC()
	if existing == nil {
		rec := models.ScanRecord{
			ScanSummary: models.ScanSummary{
				ID:         norm.ID,
				Tag:        norm.Tag,
				ComputerID: norm.ComputerID,
				Hostname:   norm.Hostname,
				DriveID:    norm.DriveID,
				RootPath:   norm.RootPath,
				HashAlgo:   norm.HashAlgo,
				Status:     models.ScanStatusInProgress,
				Version:    1,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		}
		existing = &rec
	} else {
		if !req.Resume && req.BatchIndex == 0 {
			if existing.Status != models.ScanStatusInProgress {
				m.baselines[existing.ID] = memoryBaseline{
					rootHash: existing.RootHash,
					version:  existing.Version,
					entries:  cloneEntries(existing.Entries),
					history:  cloneHistory(existing.History),
				}
			}
			existing.Entries = nil
			existing.Status = models.ScanStatusInProgress
		} else if req.Resume && existing.Status != models.ScanStatusInProgress {
			if _, hasBase := m.baselines[existing.ID]; !hasBase {
				m.baselines[existing.ID] = memoryBaseline{
					rootHash: existing.RootHash,
					version:  existing.Version,
					entries:  cloneEntries(existing.Entries),
					history:  cloneHistory(existing.History),
				}
			}
			existing.Status = models.ScanStatusInProgress
		}

		if norm.Tag != "" {
			existing.Tag = norm.Tag
		}
		if norm.Hostname != "" && norm.Hostname != "unknown-host" {
			existing.Hostname = norm.Hostname
		}
		if norm.ComputerID != "" && norm.ComputerID != "unknown-host" {
			existing.ComputerID = norm.ComputerID
		}
		if norm.DriveID != "" && norm.DriveID != "default-drive" {
			existing.DriveID = norm.DriveID
		}
		if norm.RootPath != "" {
			existing.RootPath = norm.RootPath
		}
		if req.HashAlgo != "" {
			existing.HashAlgo = req.HashAlgo
		}
		existing.UpdatedAt = now
	}

	existing.Entries, existing.TotalFiles, existing.TotalDirs, existing.TotalBytes = UpsertScanEntries(existing.Entries, req.Entries)

	if !req.Final {
		m.scans[existing.ID] = *existing
		summary := existing.ScanSummary
		return &summary, nil
	}

	rootHash := req.RootHash
	if rootHash == "" || req.Resume {
		rootHash = models.ComputeRootHash(models.HashAlgorithm(existing.HashAlgo), existing.Entries)
	}
	existing.RootHash = rootHash

	var merged models.ScanRecord
	if base, hasBase := m.baselines[existing.ID]; hasBase {
		baseRec := *existing
		baseRec.RootHash = base.rootHash
		baseRec.Version = base.version
		baseRec.Entries = base.entries
		baseRec.History = base.history
		merged = MergeScanUpdate(&baseRec, *existing)
		delete(m.baselines, existing.ID)
	} else {
		merged = MergeScanUpdate(nil, *existing)
	}
	merged.Status = models.ScanStatusCompleted
	merged.Entries = cloneEntries(merged.Entries)
	merged.History = cloneHistory(merged.History)

	m.scans[merged.ID] = merged
	summary := merged.ScanSummary
	return &summary, nil
}

func cloneEntries(src []models.ScanEntry) []models.ScanEntry {
	if src == nil {
		return nil
	}
	dst := make([]models.ScanEntry, len(src))
	copy(dst, src)
	return dst
}

// ListScans returns all stored scan summaries ordered newest update first.
func (m *MemoryStore) ListScans(_ context.Context) ([]models.ScanSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	summaries := make([]models.ScanSummary, 0, len(m.scans))
	for _, s := range m.scans {
		summaries = append(summaries, s.ScanSummary)
	}

	sort.Slice(summaries, func(i, j int) bool {
		ti := summaries[i].UpdatedAt
		if ti.IsZero() {
			ti = summaries[i].CreatedAt
		}
		tj := summaries[j].UpdatedAt
		if tj.IsZero() {
			tj = summaries[j].CreatedAt
		}
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return summaries[i].ID < summaries[j].ID
	})
	return summaries, nil
}

// ListComputers returns all tracked computers and their directory snapshots.
func (m *MemoryStore) ListComputers(ctx context.Context) ([]models.Computer, error) {
	summaries, err := m.ListScans(ctx)
	if err != nil {
		return nil, err
	}
	return GroupScansByComputer(summaries), nil
}

// GetScan retrieves a stored ScanRecord by ID (including its entries and change history).
func (m *MemoryStore) GetScan(_ context.Context, scanID string) (*models.ScanRecord, error) {
	m.mu.RLock()
	rec, ok := m.scans[scanID]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrScanNotFound, scanID)
	}

	entriesCopy := make([]models.ScanEntry, len(rec.Entries))
	copy(entriesCopy, rec.Entries)
	rec.Entries = entriesCopy

	rec.History = cloneHistory(rec.History)
	return &rec, nil
}

func cloneHistory(src []models.ScanHistoryEntry) []models.ScanHistoryEntry {
	if src == nil {
		return nil
	}
	dst := make([]models.ScanHistoryEntry, len(src))
	copy(dst, src)
	for i := range dst {
		if src[i].Changes != nil {
			changesCopy := make([]models.DiffItem, len(src[i].Changes))
			copy(changesCopy, src[i].Changes)
			dst[i].Changes = changesCopy
		}
	}
	return dst
}

// GetScanHistory retrieves the historical revisions of a stored directory snapshot by ID.
func (m *MemoryStore) GetScanHistory(ctx context.Context, scanID string) ([]models.ScanHistoryEntry, error) {
	rec, err := m.GetScan(ctx, scanID)
	if err != nil {
		return nil, err
	}
	return rec.History, nil
}

// DiffScans compares two stored scans by ID.
func (m *MemoryStore) DiffScans(ctx context.Context, baseScanID, targetScanID string) (*models.DiffReport, error) {
	baseScan, err := m.GetScan(ctx, baseScanID)
	if err != nil {
		return nil, err
	}
	targetScan, err := m.GetScan(ctx, targetScanID)
	if err != nil {
		return nil, err
	}
	return ComputeDiff(*baseScan, *targetScan), nil
}

// FindDuplicates finds duplicate files within a single scan (if scanID is non-empty)
// or across all stored scans.
func (m *MemoryStore) FindDuplicates(ctx context.Context, scanID string) ([]models.DuplicateGroup, error) {
	if scanID != "" {
		rec, err := m.GetScan(ctx, scanID)
		if err != nil {
			return nil, err
		}
		return ComputeDuplicates([]models.ScanRecord{*rec}), nil
	}

	m.mu.RLock()
	allScans := make([]models.ScanRecord, 0, len(m.scans))
	for _, s := range m.scans {
		allScans = append(allScans, s)
	}
	m.mu.RUnlock()

	return ComputeDuplicates(allScans), nil
}

// Close is a no-op for MemoryStore.
func (m *MemoryStore) Close() error {
	return nil
}
