package looker

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	var lastSnap models.ProgressSnapshot
	files, errLog := collectFiles([]string{tmpDir}, models.LookOptions{
		Recursive: true,
		Hash:      true,
		HashAlgo:  models.HashCRC32,
		Workers:   2,
		OnProgress: func(snap models.ProgressSnapshot) {
			lastSnap = snap
		},
	})
	if errLog != "" {
		t.Errorf("unexpected error output: %s", errLog)
	}
	if !lastSnap.Done || lastSnap.Completed != 3 || lastSnap.Dirs != 2 || lastSnap.Bytes != 6 {
		t.Errorf("unexpected final ProgressSnapshot: %+v", lastSnap)
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

func TestWatchDetectsAddedModifiedDeletedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	initialFile := filepath.Join(tmpDir, "a.txt")
	if err := os.WriteFile(initialFile, []byte("initial"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := models.LookOptions{
		Recursive:  true,
		ShowHidden: false,
		Hash:       true,
		HashAlgo:   models.HashCRC32,
		Live:       true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	diffs := make(chan models.DiffReport, 8)
	errCh := make(chan error, 1)

	initialFiles := Collect([]string{tmpDir}, opts, nil)
	go func() {
		errCh <- Watch(
			ctx,
			[]string{tmpDir},
			opts,
			initialFiles,
			25*time.Millisecond,
			nil,
			func(_ []models.File, _, _ models.ScanRecord, diff models.DiffReport) error {
				diffs <- diff
				return nil
			},
		)
	}()

	// Give fsnotify a brief moment to register initial directory watches.
	time.Sleep(50 * time.Millisecond)

	// 1. Write a hidden file (should be ignored because ShowHidden is false)
	// and create a visible file b.txt.
	if err := os.WriteFile(filepath.Join(tmpDir, ".ignored"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	secondFile := filepath.Join(tmpDir, "b.txt")
	if err := os.WriteFile(secondFile, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case d := <-diffs:
		if d.Added != 1 || len(d.Changes) != 1 || d.Changes[0].RelPath != "b.txt" || d.Changes[0].Change != models.DiffAdded {
			t.Fatalf("unexpected diff after adding b.txt: %+v", d)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for added file event")
	}

	// 2. Modify a.txt and remove b.txt.
	if err := os.WriteFile(initialFile, []byte("updated-content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(secondFile); err != nil {
		t.Fatal(err)
	}

	select {
	case d := <-diffs:
		if d.Modified != 1 || d.Deleted != 1 {
			t.Fatalf("expected 1 modified and 1 deleted, got %+v", d)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for modified/deleted file event")
	}

	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("Watch returned unexpected error: %v", err)
	}
}

func TestRunReusesKnownFilesWithoutRehashing(t *testing.T) {
	tmpDir := t.TempDir()
	cachedFile := filepath.Join(tmpDir, "cached.txt")
	freshFile := filepath.Join(tmpDir, "fresh.txt")

	if err := os.WriteFile(cachedFile, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(freshFile, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(cachedFile)
	if err != nil {
		t.Fatal(err)
	}

	files, errLog := collectFiles([]string{tmpDir}, models.LookOptions{
		Recursive: true,
		Hash:      true,
		HashAlgo:  models.HashCRC32,
		KnownFiles: map[string]models.KnownFileMetadata{
			cachedFile: {
				Size:    info.Size(),
				ModTime: info.ModTime(),
				Hash:    "reused-hash-from-checkpoint",
			},
		},
	})
	if errLog != "" {
		t.Fatalf("unexpected error output: %s", errLog)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	got := make(map[string]string, 2)
	for _, f := range files {
		got[f.FileInfo.Name()] = f.Hash
	}
	if got["cached.txt"] != "reused-hash-from-checkpoint" {
		t.Errorf("cached.txt hash = %q, want reused-hash-from-checkpoint", got["cached.txt"])
	}
	if got["fresh.txt"] != "352441c2" {
		t.Errorf("fresh.txt hash = %q, want 352441c2", got["fresh.txt"])
	}
}
