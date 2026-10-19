package cli

import (
	"bytes"
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alecthomas/kong"
	"github.com/vitorecomp/go-ls/pkg/models"
	"github.com/vitorecomp/go-ls/pkg/server"
	"github.com/vitorecomp/go-ls/pkg/store"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func TestParseScanSubcommandAndOptions(t *testing.T) {
	var parsed CLI
	parser, err := kong.New(&parsed)
	if err != nil {
		t.Fatalf("kong.New failed: %v", err)
	}

	ctx, err := parser.Parse([]string{"scan", "-la", "-R", "--hash", "--hash-algo=sha256", "--color=never", "--workers=8", "--server=http://localhost:8080", "/tmp"})
	if err != nil {
		t.Fatalf("parser.Parse failed: %v", err)
	}
	if ctx.Command() != "scan <path>" && ctx.Command() != "scan" {
		t.Errorf("unexpected command: %q", ctx.Command())
	}

	lookOpts := parsed.Scan.ToLookOptions()
	if !lookOpts.ShowHidden || !lookOpts.Recursive || !lookOpts.Hash {
		t.Errorf("unexpected LookOptions booleans: %+v", lookOpts)
	}
	if lookOpts.HashAlgo != models.HashSHA256 {
		t.Errorf("HashAlgo = %q, want %q", lookOpts.HashAlgo, models.HashSHA256)
	}
	if lookOpts.Workers != 8 {
		t.Errorf("Workers = %d, want 8", lookOpts.Workers)
	}

	outOpts := parsed.Scan.ToOutputOptions()
	if outOpts.Format != models.FormatLong {
		t.Errorf("Format = %q, want %q", outOpts.Format, models.FormatLong)
	}
	if !outOpts.NoColor {
		t.Errorf("expected NoColor=true when --color=never")
	}
	if parsed.Scan.ServerURL != "http://localhost:8080" {
		t.Errorf("ServerURL = %q, want http://localhost:8080", parsed.Scan.ServerURL)
	}
}

func TestParseDiffAndServeSubcommands(t *testing.T) {
	var parsed CLI
	parser, err := kong.New(&parsed)
	if err != nil {
		t.Fatalf("kong.New failed: %v", err)
	}

	ctx, err := parser.Parse([]string{"diff", "scan-1", "scan-2", "--server=http://example.com:8080"})
	if err != nil {
		t.Fatalf("parser.Parse(diff) failed: %v", err)
	}
	if ctx.Command() != "diff <base-scan-id> <target-scan-id>" {
		t.Errorf("unexpected diff command: %q", ctx.Command())
	}
	if parsed.Diff.BaseID != "scan-1" || parsed.Diff.TargetID != "scan-2" {
		t.Errorf("unexpected diff IDs: %+v", parsed.Diff)
	}

	ctx, err = parser.Parse([]string{"serve", "--listen=:9090", "--db-url=memory://", "--dev", "--ui-proxy=http://127.0.0.1:3000"})
	if err != nil {
		t.Fatalf("parser.Parse(serve) failed: %v", err)
	}
	if ctx.Command() != "serve" {
		t.Errorf("unexpected serve command: %q", ctx.Command())
	}
	if parsed.Serve.ListenAddr != ":9090" {
		t.Errorf("ListenAddr = %q, want :9090", parsed.Serve.ListenAddr)
	}
	if !parsed.Serve.Dev || parsed.Serve.UIProxyURL != "http://127.0.0.1:3000" {
		t.Errorf("expected Dev=true and UIProxyURL=http://127.0.0.1:3000, got %+v", parsed.Serve)
	}
	if !shouldReloadGoFile("pkg/server/server.go") || shouldReloadGoFile("pkg/server/server_test.go") {
		t.Errorf("unexpected shouldReloadGoFile behavior")
	}
}

func TestParseLiveFlagEnablesLiveAndHash(t *testing.T) {
	var parsed CLI
	parser, err := kong.New(&parsed)
	if err != nil {
		t.Fatalf("kong.New failed: %v", err)
	}

	_, err = parser.Parse(NormalizeArgs([]string{"--live", "-R", "/tmp"}))
	if err != nil {
		t.Fatalf("parser.Parse(--live) failed: %v", err)
	}
	if !parsed.Scan.Live {
		t.Errorf("expected Scan.Live = true")
	}

	lookOpts := parsed.Scan.ToLookOptions()
	if !lookOpts.Live || !lookOpts.Hash {
		t.Errorf("expected LookOptions.Live=true and Hash=true, got %+v", lookOpts)
	}

	outOpts := parsed.Scan.ToOutputOptions()
	if !outOpts.Hash {
		t.Errorf("expected OutputOptions.Hash=true when --live is enabled, got %+v", outOpts)
	}
}

func TestScanCmdLiveMonitoringAndServerUpload(t *testing.T) {
	memStore := store.NewMemoryStore()
	srv := httptest.NewServer(server.New(memStore))
	defer srv.Close()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "watched.txt")
	if err := os.WriteFile(filePath, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := ScanCmd{
		Recursive: true,
		Live:      true,
		Color:     "never",
		ServerURL: srv.URL,
		ScanID:    "live-snap",
		Quiet:     true,
		Paths:     []string{tmpDir},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var outBuf, errBuf syncBuffer
	done := make(chan error, 1)
	go func() {
		done <- cmd.RunWithContext(ctx, &outBuf, &errBuf)
	}()

	// Wait until the initial scan is stored and watching has begun.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(errBuf.String(), "Watching") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)

	// Modify watched.txt to trigger a live event and server snapshot upload.
	if err := os.WriteFile(filePath, []byte("v2-modified"), 0o644); err != nil {
		t.Fatal(err)
	}

	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(outBuf.String(), "live-snap (v2)") {
			break
		}
		time.Sleep(30 * time.Millisecond)
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("RunWithContext returned error: %v", err)
	}

	outStr := outBuf.String()
	if !strings.Contains(outStr, "~ [MODIFIED]") || !strings.Contains(outStr, "watched.txt") {
		t.Errorf("expected live modified output for watched.txt, got:\n%s", outStr)
	}

	scans, err := memStore.ListScans(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(scans) != 1 || scans[0].Version != 2 {
		t.Fatalf("expected 1 updated directory snapshot at Version=2, got %+v", scans)
	}

	history, err := memStore.GetScanHistory(context.Background(), "live-snap")
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 history revisions for live-snap, got %d", len(history))
	}
}

func TestScansDiffAndDuplicatesCmdRun(t *testing.T) {
	ctx := context.Background()
	memStore := store.NewMemoryStore()
	srv := httptest.NewServer(server.New(memStore))
	defer srv.Close()

	scanA := models.ScanRecord{
		ScanSummary: models.ScanSummary{
			ID:         "scan-a",
			ComputerID: "comp-1",
			Hostname:   "host-a",
			DriveID:    "drive-1",
			RootPath:   "/srv/a",
			HashAlgo:   "crc32",
			RootHash:   "11111111",
		},
		Entries: []models.ScanEntry{
			{RelPath: "shared.txt", AbsPath: "/srv/a/shared.txt", Name: "shared.txt", Size: 64, Hash: "dup-hash"},
			{RelPath: "only-a.txt", AbsPath: "/srv/a/only-a.txt", Name: "only-a.txt", Size: 32, Hash: "hash-a"},
		},
	}
	scanB := models.ScanRecord{
		ScanSummary: models.ScanSummary{
			ID:         "scan-b",
			ComputerID: "comp-2",
			Hostname:   "host-b",
			DriveID:    "drive-2",
			RootPath:   "/srv/b",
			HashAlgo:   "crc32",
			RootHash:   "22222222",
		},
		Entries: []models.ScanEntry{
			{RelPath: "shared.txt", AbsPath: "/srv/b/shared.txt", Name: "shared.txt", Size: 64, Hash: "dup-hash"},
			{RelPath: "copy.txt", AbsPath: "/srv/b/copy.txt", Name: "copy.txt", Size: 64, Hash: "dup-hash"},
			{RelPath: "only-b.txt", AbsPath: "/srv/b/only-b.txt", Name: "only-b.txt", Size: 48, Hash: "hash-b"},
		},
	}

	if _, err := memStore.SaveScan(ctx, scanA); err != nil {
		t.Fatalf("SaveScan(scanA) failed: %v", err)
	}
	if _, err := memStore.SaveScan(ctx, scanB); err != nil {
		t.Fatalf("SaveScan(scanB) failed: %v", err)
	}

	// 1. ScansCmd.Run (list all scans + inspect single scan)
	listCmd := ScansCmd{ServerURL: srv.URL, JSON: true}
	if err := listCmd.Run(); err != nil {
		t.Fatalf("ScansCmd.Run (list) failed: %v", err)
	}

	inspectCmd := ScansCmd{ServerURL: srv.URL, ScanID: "scan-a", JSON: false}
	if err := inspectCmd.Run(); err != nil {
		t.Fatalf("ScansCmd.Run (inspect scan-a) failed: %v", err)
	}

	// 2. DiffCmd.Run (compare scan-a and scan-b)
	diffCmd := DiffCmd{
		ServerURL: srv.URL,
		BaseID:    "scan-a",
		TargetID:  "scan-b",
		Color:     "never",
		JSON:      false,
	}
	if err := diffCmd.Run(); err != nil {
		t.Fatalf("DiffCmd.Run failed: %v", err)
	}

	// 3. DuplicatesCmd.Run (global and per-scan)
	dupGlobalCmd := DuplicatesCmd{ServerURL: srv.URL, JSON: false}
	if err := dupGlobalCmd.Run(); err != nil {
		t.Fatalf("DuplicatesCmd.Run (global) failed: %v", err)
	}

	dupSingleCmd := DuplicatesCmd{ServerURL: srv.URL, ScanID: "scan-b", JSON: true}
	if err := dupSingleCmd.Run(); err != nil {
		t.Fatalf("DuplicatesCmd.Run (scan-b) failed: %v", err)
	}
}
