package looker

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitorecomp/go-ls/pkg/models"
)

func collectFiles(paths []string, opts models.LookOptions) ([]models.File, string) {
	out := make(chan models.File, 64)
	var errBuf bytes.Buffer
	go Run(paths, opts, out, &errBuf)

	var files []models.File
	for f := range out {
		files = append(files, f)
	}
	return files, errBuf.String()
}

func TestRunFlatAndHiddenFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "visible.txt"), []byte("v"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("h"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Without ShowHidden
	files, errLog := collectFiles([]string{tmpDir}, models.LookOptions{ShowHidden: false})
	if errLog != "" {
		t.Errorf("unexpected error output: %s", errLog)
	}
	if len(files) != 1 || files[0].FileInfo.Name() != "visible.txt" {
		t.Errorf("expected only visible.txt, got %+v", files)
	}

	// With ShowHidden
	filesAll, _ := collectFiles([]string{tmpDir}, models.LookOptions{ShowHidden: true})
	if len(filesAll) != 2 {
		t.Errorf("expected 2 files with ShowHidden=true, got %d", len(filesAll))
	}
}

func TestRunRecursiveAndDirectoryHashing(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "nested")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "root.txt"), []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "child.txt"), []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, errLog := collectFiles([]string{tmpDir}, models.LookOptions{
		Recursive: true,
		Hash:      true,
		HashAlgo:  models.HashCRC32,
		Workers:   2,
	})
	if errLog != "" {
		t.Errorf("unexpected error output: %s", errLog)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 entries (root.txt, nested, child.txt), got %d", len(files))
	}

	var expectedDir models.File
	wantNestedHash := expectedDir.ComputeDirHash(models.HashCRC32, []string{"352441c2"})

	for _, f := range files {
		if !f.Dir && f.Hash != "352441c2" {
			t.Errorf("expected file hash 352441c2 for %s, got %q", f.FileInfo.Name(), f.Hash)
		}
		if f.Dir && f.Hash != wantNestedHash {
			t.Errorf("expected folder hash %q for %s, got %q", wantNestedHash, f.FileInfo.Name(), f.Hash)
		}
	}
}

func TestNonRecursiveStillComputesFolderHashWhenHashEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "inside.txt"), []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Recursive is false, Hash is true: only "sub" should be emitted, but its Hash must be computed from inside.txt!
	files, errLog := collectFiles([]string{tmpDir}, models.LookOptions{
		Recursive: false,
		Hash:      true,
		HashAlgo:  models.HashCRC32,
	})
	if errLog != "" {
		t.Errorf("unexpected error output: %s", errLog)
	}
	if len(files) != 1 || !files[0].Dir || files[0].FileInfo.Name() != "sub" {
		t.Fatalf("expected only 'sub' directory entry, got %+v", files)
	}

	var expectedDir models.File
	wantSubHash := expectedDir.ComputeDirHash(models.HashCRC32, []string{"352441c2"})
	if files[0].Hash != wantSubHash {
		t.Errorf("sub folder hash = %q, want %q", files[0].Hash, wantSubHash)
	}
}

func TestRunSingleFileArgument(t *testing.T) {
	tmpDir := t.TempDir()
	singleFile := filepath.Join(tmpDir, "single.txt")
	if err := os.WriteFile(singleFile, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, errLog := collectFiles([]string{singleFile}, models.LookOptions{
		Hash:     true,
		HashAlgo: models.HashCRC32,
	})
	if errLog != "" {
		t.Errorf("unexpected error output: %s", errLog)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].GetAbsolutePath() != singleFile {
		t.Errorf("path = %q, want %q", files[0].GetAbsolutePath(), singleFile)
	}
	if files[0].Hash != "352441c2" {
		t.Errorf("hash = %q, want 352441c2", files[0].Hash)
	}
}

func TestRunNonExistentPathReportsErrorGracefully(t *testing.T) {
	files, errLog := collectFiles([]string{"/non/existent/path/for/go-ls"}, models.LookOptions{})
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
	if !strings.Contains(errLog, "cannot access") {
		t.Errorf("expected 'cannot access' error message, got %q", errLog)
	}
}
