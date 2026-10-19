package models

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNewFileAndMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "sample.txt")
	content := []byte("hello go-ls\n")
	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("failed to stat test file: %v", err)
	}

	f := NewFile(tmpDir, info)
	if f.Dir {
		t.Errorf("expected Dir=false, got true")
	}
	if got := f.GetAbsolutePath(); got != filePath {
		t.Errorf("GetAbsolutePath() = %q, want %q", got, filePath)
	}
	if f.Owner.Username == "" {
		t.Errorf("expected Owner.Username to be populated")
	}
	if f.Group.Name == "" {
		t.Errorf("expected Group.Name to be populated")
	}
	if f.UpdatedAt == "" || f.AccessAt == "" || f.CreatedAt == "" {
		t.Errorf("expected timestamps to be populated, got UpdatedAt=%q AccessAt=%q CreatedAt=%q",
			f.UpdatedAt, f.AccessAt, f.CreatedAt)
	}
}

func TestComputeHash(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "data.txt")
	if err := os.WriteFile(filePath, []byte("abc"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("failed to stat test file: %v", err)
	}

	f := NewFile(tmpDir, info)

	// CRC32 of "abc" is 352441c2
	if err := f.ComputeHash(HashCRC32); err != nil {
		t.Fatalf("ComputeHash(HashCRC32) failed: %v", err)
	}
	if f.Hash != "352441c2" {
		t.Errorf("CRC32 hash = %q, want %q", f.Hash, "352441c2")
	}

	// SHA256 of "abc" is ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad
	wantSHA := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if err := f.ComputeHash(HashSHA256); err != nil {
		t.Fatalf("ComputeHash(HashSHA256) failed: %v", err)
	}
	if f.Hash != wantSHA {
		t.Errorf("SHA256 hash = %q, want %q", f.Hash, wantSHA)
	}
}

func TestComputeHashOnDirectoryReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	info, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("failed to stat temp dir: %v", err)
	}

	f := NewFile(filepath.Dir(tmpDir), info)
	err = f.ComputeHash(HashCRC32)
	if !errors.Is(err, ErrDirectoryChecksum) {
		t.Errorf("ComputeHash on directory returned %v, want %v", err, ErrDirectoryChecksum)
	}
}

func TestComputeDirHash(t *testing.T) {
	tmpDir := t.TempDir()
	info, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("failed to stat temp dir: %v", err)
	}

	dirFile := NewFile(filepath.Dir(tmpDir), info)
	childHashes := []string{"352441c2", "02821b30"}

	gotCRC := dirFile.ComputeDirHash(HashCRC32, childHashes)
	if gotCRC == "" || dirFile.Hash != gotCRC {
		t.Fatalf("expected non-empty CRC32 dir hash, got %q", gotCRC)
	}

	// Verify determinism and sensitivity to child hash changes
	dirFile2 := NewFile(filepath.Dir(tmpDir), info)
	if got2 := dirFile2.ComputeDirHash(HashCRC32, childHashes); got2 != gotCRC {
		t.Errorf("ComputeDirHash not deterministic: %q != %q", got2, gotCRC)
	}

	dirFileChanged := NewFile(filepath.Dir(tmpDir), info)
	if gotChanged := dirFileChanged.ComputeDirHash(HashCRC32, []string{"352441c2", "ffffffff"}); gotChanged == gotCRC {
		t.Errorf("expected different dir hash when child hash changes, got %q", gotChanged)
	}
}

func TestFileMarshalJSON(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "item.txt")
	if err := os.WriteFile(filePath, []byte("json test"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("failed to stat test file: %v", err)
	}

	f := NewFile(tmpDir, info)
	_ = f.ComputeHash(HashCRC32)

	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded FileJSON
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if decoded.Name != "item.txt" {
		t.Errorf("decoded.Name = %q, want %q", decoded.Name, "item.txt")
	}
	if decoded.Path != filePath {
		t.Errorf("decoded.Path = %q, want %q", decoded.Path, filePath)
	}
	if decoded.Type != "file" || decoded.IsDir {
		t.Errorf("unexpected type/is_dir: %+v", decoded)
	}
	if decoded.Hash == "" {
		t.Errorf("expected non-empty hash in JSON output")
	}
}

func TestDetectComputerAndDriveID(t *testing.T) {
	tmpDir := t.TempDir()
	compID := DetectComputerID("test-host")
	if compID == "" {
		t.Errorf("expected non-empty ComputerID")
	}

	driveID := DetectDriveID(tmpDir)
	if driveID == "" || driveID == "unknown-drive" {
		t.Errorf("expected valid DriveID for %s, got %q", tmpDir, driveID)
	}
}
