package server_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitorecomp/go-ls/pkg/client"
	"github.com/vitorecomp/go-ls/pkg/models"
	"github.com/vitorecomp/go-ls/pkg/server"
	"github.com/vitorecomp/go-ls/pkg/store"
)

func TestClientServerIntegration(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	srv := server.New(st)

	ts := httptest.NewServer(srv)
	defer ts.Close()

	cli := client.New(ts.URL)

	if err := cli.Health(ctx); err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	scan1 := models.ScanRecord{
		ScanSummary: models.ScanSummary{
			ID:       "scan-1",
			Tag:      "before",
			Hostname: "host-1",
			RootPath: "/var/data",
			HashAlgo: "crc32",
			RootHash: "10000001",
		},
		Entries: []models.ScanEntry{
			{RelPath: "a.txt", AbsPath: "/var/data/a.txt", Name: "a.txt", Size: 100, Hash: "h1"},
			{RelPath: "b.txt", AbsPath: "/var/data/b.txt", Name: "b.txt", Size: 200, Hash: "h2"},
		},
	}

	scan2 := models.ScanRecord{
		ScanSummary: models.ScanSummary{
			ID:       "scan-2",
			Tag:      "after",
			Hostname: "host-2",
			RootPath: "/mnt/backup",
			HashAlgo: "crc32",
			RootHash: "10000002",
		},
		Entries: []models.ScanEntry{
			{RelPath: "a.txt", AbsPath: "/mnt/backup/a.txt", Name: "a.txt", Size: 100, Hash: "h1"},
			{RelPath: "b.txt", AbsPath: "/mnt/backup/b.txt", Name: "b.txt", Size: 250, Hash: "h2-mod"},
			{RelPath: "c.txt", AbsPath: "/mnt/backup/c.txt", Name: "c.txt", Size: 100, Hash: "h1"}, // duplicate of a.txt
		},
	}

	sum1, err := cli.UploadScan(ctx, scan1)
	if err != nil {
		t.Fatalf("UploadScan(scan1) failed: %v", err)
	}
	if sum1.ID != "scan-1" || sum1.TotalFiles != 2 {
		t.Errorf("unexpected sum1: %+v", sum1)
	}

	if _, err := cli.UploadScan(ctx, scan2); err != nil {
		t.Fatalf("UploadScan(scan2) failed: %v", err)
	}

	scans, err := cli.ListScans(ctx)
	if err != nil || len(scans) != 2 {
		t.Fatalf("ListScans failed: len=%d err=%v", len(scans), err)
	}

	gotScan1, err := cli.GetScan(ctx, "scan-1")
	if err != nil || len(gotScan1.Entries) != 2 {
		t.Fatalf("GetScan(scan-1) failed: %v", err)
	}

	diff, err := cli.DiffScans(ctx, "scan-1", "scan-2")
	if err != nil {
		t.Fatalf("DiffScans failed: %v", err)
	}
	if diff.Added != 1 || diff.Modified != 1 || diff.Deleted != 0 || diff.Unchanged != 1 {
		t.Errorf("unexpected diff summary: %+v", diff)
	}

	dups, err := cli.FindDuplicates(ctx, "scan-2")
	if err != nil {
		t.Fatalf("FindDuplicates failed: %v", err)
	}
	if len(dups) != 1 || dups[0].Count != 2 {
		t.Errorf("expected 1 duplicate group of count 2 in scan-2, got %+v", dups)
	}

	// Re-scanning the same directory on host-1 updates scan-1 in place and records change history.
	scan1Update := models.ScanRecord{
		ScanSummary: models.ScanSummary{
			Hostname: "host-1",
			RootPath: "/var/data",
			HashAlgo: "crc32",
			RootHash: "10000099",
		},
		Entries: []models.ScanEntry{
			{RelPath: "a.txt", AbsPath: "/var/data/a.txt", Name: "a.txt", Size: 150, Hash: "h1-updated"},
		},
	}
	updatedSum, err := cli.UploadScan(ctx, scan1Update)
	if err != nil {
		t.Fatalf("UploadScan(scan1Update) failed: %v", err)
	}
	if updatedSum.ID != "scan-1" || updatedSum.Version != 2 {
		t.Errorf("expected scan-1 updated to Version 2, got %+v", updatedSum)
	}

	computers, err := cli.ListComputers(ctx)
	if err != nil || len(computers) != 2 {
		t.Fatalf("ListComputers failed: len=%d err=%v", len(computers), err)
	}

	history, err := cli.GetScanHistory(ctx, "scan-1")
	if err != nil || len(history) != 2 {
		t.Fatalf("GetScanHistory(scan-1) failed: len=%d err=%v", len(history), err)
	}
	if history[0].Version != 2 || history[0].Modified != 1 || history[0].Deleted != 1 {
		t.Errorf("unexpected latest history entry for scan-1: %+v", history[0])
	}
}

func TestServerServesStaticFrontend(t *testing.T) {
	uiDir := t.TempDir()
	indexHTML := `<!DOCTYPE html><html><head><title>go-ls UI</title></head><body><h1>go-ls</h1></body></html>`
	if err := os.WriteFile(filepath.Join(uiDir, "index.html"), []byte(indexHTML), 0o644); err != nil {
		t.Fatal(err)
	}

	st := store.NewMemoryStore()
	srv := server.NewWithUI(st, uiDir)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "go-ls UI") {
		t.Errorf("expected body to contain 'go-ls UI', got %q", string(body))
	}
}

func TestServerProxiesDevUIAndHandlesAPI(t *testing.T) {
	mockNextDev := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><body>HMR Dev Server</body></html>`))
	}))
	defer mockNextDev.Close()

	st := store.NewMemoryStore()
	srv := server.NewWithUIAndProxy(st, "", mockNextDev.URL)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// Root request should be reverse-proxied to the Next.js dev server
	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "HMR Dev Server") {
		t.Errorf("expected proxied HMR dev server response, got %q", string(body))
	}

	// API requests should still be served directly by the Go server
	healthResp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz failed: %v", err)
	}
	defer healthResp.Body.Close()
	if healthResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK on /healthz, got %d", healthResp.StatusCode)
	}
}
