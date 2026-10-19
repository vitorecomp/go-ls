package store

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/vitorecomp/go-ls/pkg/models"
)

// MemoryStore is a thread-safe in-memory implementation of Store.
type MemoryStore struct {
	mu    sync.RWMutex
	scans map[string]models.ScanRecord
}

// NewMemoryStore creates an initialized MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		scans: make(map[string]models.ScanRecord),
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

	entriesCopy := make([]models.ScanEntry, len(merged.Entries))
	copy(entriesCopy, merged.Entries)
	merged.Entries = entriesCopy

	merged.History = cloneHistory(merged.History)

	m.scans[merged.ID] = merged

	summary := merged.ScanSummary
	return &summary, nil
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
