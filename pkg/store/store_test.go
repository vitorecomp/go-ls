package store

import (
	"context"
	"errors"
	"testing"

	"github.com/vitorecomp/go-ls/pkg/models"
)

func TestMemoryStoreSaveListDiffAndDuplicates(t *testing.T) {
	ctx := context.Background()
	st, err := Open(ctx, "memory://")
	if err != nil {
		t.Fatalf("Open(memory://) failed: %v", err)
	}
	defer st.Close()

	baseScan := models.ScanRecord{
		ScanSummary: models.ScanSummary{
			ID:       "scan-base",
			Hostname: "node-1",
			RootPath: "/data/source",
			HashAlgo: "crc32",
			RootHash: "11111111",
		},
		Entries: []models.ScanEntry{
			{RelPath: "keep.txt", AbsPath: "/data/source/keep.txt", Name: "keep.txt", Size: 10, Hash: "aaaa"},
			{RelPath: "modify.txt", AbsPath: "/data/source/modify.txt", Name: "modify.txt", Size: 20, Hash: "bbbb"},
			{RelPath: "delete.txt", AbsPath: "/data/source/delete.txt", Name: "delete.txt", Size: 30, Hash: "cccc"},
		},
	}

	targetScan := models.ScanRecord{
		ScanSummary: models.ScanSummary{
			ID:       "scan-target",
			Hostname: "node-2",
			RootPath: "/data/backup",
			HashAlgo: "crc32",
			RootHash: "22222222",
		},
		Entries: []models.ScanEntry{
			{RelPath: "keep.txt", AbsPath: "/data/backup/keep.txt", Name: "keep.txt", Size: 10, Hash: "aaaa"},
			{RelPath: "modify.txt", AbsPath: "/data/backup/modify.txt", Name: "modify.txt", Size: 25, Hash: "bbbb-new"},
			{RelPath: "added.txt", AbsPath: "/data/backup/added.txt", Name: "added.txt", Size: 10, Hash: "aaaa"}, // duplicate of keep.txt
		},
	}

	if _, err := st.SaveScan(ctx, baseScan); err != nil {
		t.Fatalf("SaveScan(baseScan) failed: %v", err)
	}
	if _, err := st.SaveScan(ctx, targetScan); err != nil {
		t.Fatalf("SaveScan(targetScan) failed: %v", err)
	}

	list, err := st.ListScans(ctx)
	if err != nil || len(list) != 2 {
		t.Fatalf("ListScans returned %d scans, err=%v", len(list), err)
	}

	diff, err := st.DiffScans(ctx, "scan-base", "scan-target")
	if err != nil {
		t.Fatalf("DiffScans failed: %v", err)
	}
	if diff.Identical {
		t.Errorf("expected Identical=false")
	}
	if diff.Added != 1 || diff.Deleted != 1 || diff.Modified != 1 || diff.Unchanged != 1 {
		t.Errorf("unexpected diff counts: %+v", diff)
	}

	// Duplicates within scan-target (keep.txt and added.txt both have hash "aaaa" and size 10)
	dups, err := st.FindDuplicates(ctx, "scan-target")
	if err != nil {
		t.Fatalf("FindDuplicates failed: %v", err)
	}
	if len(dups) != 1 || dups[0].Count != 2 || dups[0].WastedBytes != 10 {
		t.Errorf("unexpected duplicate groups for scan-target: %+v", dups)
	}

	// Missing scan returns ErrScanNotFound
	_, err = st.GetScan(ctx, "does-not-exist")
	if !errors.Is(err, ErrScanNotFound) {
		t.Errorf("expected ErrScanNotFound, got %v", err)
	}
}

func TestSaveScanUpdatesExistingComputerDirectoryAndTracksHistory(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryStore()
	defer st.Close()

	// Initial scan of /srv/photos on workstation-1, drive uuid:disk-1
	initial := models.ScanRecord{
		ScanSummary: models.ScanSummary{
			ComputerID: "comp-1",
			Hostname:   "workstation-1",
			DriveID:    "uuid:disk-1",
			RootPath:   "/srv/photos",
			HashAlgo:   "crc32",
			RootHash:   "10000001",
		},
		Entries: []models.ScanEntry{
			{RelPath: "img1.jpg", AbsPath: "/srv/photos/img1.jpg", Name: "img1.jpg", Size: 1000, Hash: "h1"},
			{RelPath: "img2.jpg", AbsPath: "/srv/photos/img2.jpg", Name: "img2.jpg", Size: 2000, Hash: "h2"},
		},
	}

	sum1, err := st.SaveScan(ctx, initial)
	if err != nil {
		t.Fatalf("SaveScan(initial) failed: %v", err)
	}
	if sum1.Version != 1 {
		t.Fatalf("expected Version=1, got %d", sum1.Version)
	}

	// Second scan of the SAME directory (/srv/photos) on the SAME computer/drive (with no explicit ID)
	updated := models.ScanRecord{
		ScanSummary: models.ScanSummary{
			ComputerID: "comp-1",
			Hostname:   "workstation-1",
			DriveID:    "uuid:disk-1",
			RootPath:   "/srv/photos",
			HashAlgo:   "crc32",
			RootHash:   "10000002",
			Tag:        "v2-update",
		},
		Entries: []models.ScanEntry{
			{RelPath: "img1.jpg", AbsPath: "/srv/photos/img1.jpg", Name: "img1.jpg", Size: 1200, Hash: "h1-mod"},
			{RelPath: "img3.jpg", AbsPath: "/srv/photos/img3.jpg", Name: "img3.jpg", Size: 3000, Hash: "h3"},
		},
	}

	sum2, err := st.SaveScan(ctx, updated)
	if err != nil {
		t.Fatalf("SaveScan(updated) failed: %v", err)
	}
	if sum2.ID != sum1.ID {
		t.Errorf("expected same directory snapshot ID %q on update, got %q", sum1.ID, sum2.ID)
	}
	if sum2.Version != 2 {
		t.Errorf("expected Version=2 after update, got %d", sum2.Version)
	}

	// Verify only 1 active snapshot exists for this directory, contained inside 1 Computer
	scans, err := st.ListScans(ctx)
	if err != nil || len(scans) != 1 {
		t.Fatalf("expected 1 active directory snapshot, got %d (err=%v)", len(scans), err)
	}

	computers, err := st.ListComputers(ctx)
	if err != nil || len(computers) != 1 {
		t.Fatalf("expected 1 computer, got %d (err=%v)", len(computers), err)
	}
	if len(computers[0].Directories) != 1 || len(computers[0].Drives) != 1 || computers[0].Drives[0] != "uuid:disk-1" {
		t.Errorf("unexpected computer structure: %+v", computers[0])
	}

	// Verify directory history contains both v2 (latest diff) and v1 (initial)
	history, err := st.GetScanHistory(ctx, sum1.ID)
	if err != nil {
		t.Fatalf("GetScanHistory failed: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 history revisions, got %d", len(history))
	}
	if history[0].Version != 2 || history[0].Added != 1 || history[0].Modified != 1 || history[0].Deleted != 1 {
		t.Errorf("unexpected v2 history entry: %+v", history[0])
	}
	if history[0].PreviousHash != "10000001" || history[0].RootHash != "10000002" {
		t.Errorf("unexpected v2 hash transition: %s -> %s", history[0].PreviousHash, history[0].RootHash)
	}
	if history[1].Version != 1 || history[1].Added != 2 {
		t.Errorf("unexpected v1 history entry: %+v", history[1])
	}

	// Verify deep copy of ScanHistoryEntry.Changes prevents external mutation of internal store state
	if len(history[0].Changes) == 0 {
		t.Fatalf("expected non-empty Changes on v2 history entry")
	}
	origRelPath := history[0].Changes[0].RelPath
	history[0].Changes[0].RelPath = "mutated-by-caller.jpg"

	freshHistory, err := st.GetScanHistory(ctx, sum1.ID)
	if err != nil {
		t.Fatalf("GetScanHistory after mutation failed: %v", err)
	}
	if freshHistory[0].Changes[0].RelPath != origRelPath {
		t.Errorf("MemoryStore internal history slice was mutated: got %q, want %q", freshHistory[0].Changes[0].RelPath, origRelPath)
	}
}
