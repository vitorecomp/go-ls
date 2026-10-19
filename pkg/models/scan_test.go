package models

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildScanRecord(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "alpha.txt")
	if err := os.WriteFile(filePath, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	fInfo, err := os.Stat(filePath)
	if err != nil {
		t.Fatal(err)
	}
	dInfo, err := os.Stat(subDir)
	if err != nil {
		t.Fatal(err)
	}

	f := NewFile(tmpDir, fInfo)
	_ = f.ComputeHash(HashCRC32)

	d := NewFile(tmpDir, dInfo)
	d.ComputeDirHash(HashCRC32, nil)

	rec := BuildScanRecord("scan-1", "daily", "host-a", tmpDir, HashCRC32, []File{d, f})
	if rec.ID != "scan-1" || rec.Tag != "daily" || rec.Hostname != "host-a" {
		t.Errorf("unexpected summary metadata: %+v", rec.ScanSummary)
	}
	if rec.TotalFiles != 1 || rec.TotalDirs != 1 || rec.TotalBytes != 3 {
		t.Errorf("unexpected counts: files=%d dirs=%d bytes=%d", rec.TotalFiles, rec.TotalDirs, rec.TotalBytes)
	}
	if rec.RootHash == "" {
		t.Errorf("expected non-empty RootHash")
	}
	if len(rec.Entries) != 2 || rec.Entries[0].RelPath != "alpha.txt" {
		t.Errorf("expected sorted entries starting with alpha.txt, got %+v", rec.Entries)
	}
}
