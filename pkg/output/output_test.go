package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitorecomp/go-ls/pkg/models"
)

func createSampleFile(t *testing.T, dir, name, content string) models.File {
	t.Helper()
	fullPath := filepath.Join(dir, name)
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", fullPath, err)
	}
	info, err := os.Stat(fullPath)
	if err != nil {
		t.Fatalf("failed to stat %s: %v", fullPath, err)
	}
	f := models.NewFile(dir, info)
	_ = f.ComputeHash(models.HashCRC32)
	return f
}

func TestWriteOutputDefaultFormat(t *testing.T) {
	tmpDir := t.TempDir()
	f := createSampleFile(t, tmpDir, "hello.txt", "abc")

	ch := make(chan models.File, 1)
	ch <- f
	close(ch)

	var buf bytes.Buffer
	WriteOutput(&buf, ch, models.OutputOptions{
		Format:  models.FormatDefault,
		NoColor: true,
	})

	got := buf.String()
	if !strings.Contains(got, "hello.txt") {
		t.Errorf("Default output = %q, expected it to contain 'hello.txt'", got)
	}
}

func TestWriteOutputLongFormat(t *testing.T) {
	tmpDir := t.TempDir()
	f := createSampleFile(t, tmpDir, "report.txt", "abc")

	ch := make(chan models.File, 1)
	ch <- f
	close(ch)

	var buf bytes.Buffer
	WriteOutput(&buf, ch, models.OutputOptions{
		Format:  models.FormatLong,
		Hash:    true,
		NoColor: true,
	})

	got := buf.String()
	if !strings.Contains(got, "report.txt") || !strings.Contains(got, "352441c2") {
		t.Errorf("Long output = %q, expected filename and CRC32 hash", got)
	}
}

func TestWriteOutputCompareFormatEscapesCommas(t *testing.T) {
	tmpDir := t.TempDir()
	f := createSampleFile(t, tmpDir, "file,with,commas.txt", "abc")

	ch := make(chan models.File, 1)
	ch <- f
	close(ch)

	var buf bytes.Buffer
	WriteOutput(&buf, ch, models.OutputOptions{
		Format: models.FormatCompare,
		Hash:   true,
	})

	records, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatalf("csv.ReadAll failed: %v", err)
	}
	if len(records) != 1 || len(records[0]) != 4 {
		t.Fatalf("expected 1 CSV row with 4 columns, got %+v", records)
	}
	if records[0][0] != "352441c2" {
		t.Errorf("hash column = %q, want 352441c2", records[0][0])
	}
	if records[0][2] != "file" {
		t.Errorf("type column = %q, want file", records[0][2])
	}
	if !strings.HasSuffix(records[0][3], "file,with,commas.txt") {
		t.Errorf("path column = %q, expected suffix 'file,with,commas.txt'", records[0][3])
	}
}

func TestWriteOutputJSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	f := createSampleFile(t, tmpDir, "data.json", "abc")

	ch := make(chan models.File, 1)
	ch <- f
	close(ch)

	var buf bytes.Buffer
	WriteOutput(&buf, ch, models.OutputOptions{
		Format: models.FormatJSON,
		Hash:   true,
	})

	var decoded models.FileJSON
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to unmarshal JSON output: %v", err)
	}
	if decoded.Name != "data.json" || decoded.Hash != "352441c2" {
		t.Errorf("unexpected JSON output: %+v", decoded)
	}
}

func TestWriteDiffAndDuplicatesReports(t *testing.T) {
	var diffBuf bytes.Buffer
	WriteDiffReport(&diffBuf, models.DiffReport{
		BaseScan:   models.ScanSummary{ID: "s1", Hostname: "h1", RootPath: "/a"},
		TargetScan: models.ScanSummary{ID: "s2", Hostname: "h2", RootPath: "/b"},
		Added:      1,
		Changes: []models.DiffItem{
			{RelPath: "new.txt", Change: models.DiffAdded, TargetHash: "1234", TargetSize: 42},
		},
	}, false, true)
	if !strings.Contains(diffBuf.String(), "+ [ADDED]") || !strings.Contains(diffBuf.String(), "new.txt") {
		t.Errorf("unexpected diff output: %s", diffBuf.String())
	}

	var dupBuf bytes.Buffer
	WriteDuplicates(&dupBuf, []models.DuplicateGroup{
		{
			Hash:        "deadbeef",
			Size:        100,
			Count:       2,
			WastedBytes: 100,
			Files: []models.DuplicateFileRef{
				{ScanID: "s1", Hostname: "h1", AbsPath: "/a/1.txt", RelPath: "1.txt"},
				{ScanID: "s1", Hostname: "h1", AbsPath: "/a/2.txt", RelPath: "2.txt"},
			},
		},
	}, false)
	if !strings.Contains(dupBuf.String(), "deadbeef") || !strings.Contains(dupBuf.String(), "Wasted: 100 bytes") {
		t.Errorf("unexpected duplicates output: %s", dupBuf.String())
	}

	var liveBuf bytes.Buffer
	WriteLiveEvent(&liveBuf, models.LiveEvent{
		RootPath: "/a",
		RootHash: "cafebabe",
		Added:    1,
		Modified: 1,
		ScanID:   "scan-live-2",
		Changes: []models.DiffItem{
			{RelPath: "created.txt", Change: models.DiffAdded, TargetHash: "1111", TargetSize: 10},
			{RelPath: "edited.txt", Change: models.DiffModified, BaseHash: "2222", TargetHash: "3333", BaseSize: 5, TargetSize: 8},
		},
	}, false, true)
	liveText := liveBuf.String()
	if !strings.Contains(liveText, "Change detected (+1 added, -0 deleted, ~1 modified)") ||
		!strings.Contains(liveText, "+ [ADDED]") ||
		!strings.Contains(liveText, "~ [MODIFIED]") ||
		!strings.Contains(liveText, "Scan stored on server: scan-live-2") {
		t.Errorf("unexpected live event output: %s", liveText)
	}

	var liveJSONBuf bytes.Buffer
	WriteLiveEvent(&liveJSONBuf, models.LiveEvent{
		RootPath: "/a",
		RootHash: "cafebabe",
		Added:    1,
	}, true, true)
	var decodedLive models.LiveEvent
	if err := json.Unmarshal(liveJSONBuf.Bytes(), &decodedLive); err != nil {
		t.Fatalf("failed to unmarshal live JSON event: %v", err)
	}
	if decodedLive.RootHash != "cafebabe" || decodedLive.Added != 1 {
		t.Errorf("unexpected decoded live event: %+v", decodedLive)
	}
}

func TestProgressBarRenderAndStdoutCoordination(t *testing.T) {
	var errBuf, outBuf bytes.Buffer
	pb := NewProgressBar(&errBuf, true)
	wrappedOut := pb.WrapStdout(&outBuf)

	pb.Update(models.ProgressSnapshot{
		Discovered: 10,
		Completed:  5,
		Dirs:       2,
		Bytes:      2048,
		Current:    "/tmp/sample/file.go",
	})

	if _, err := wrappedOut.Write([]byte("file.go\n")); err != nil {
		t.Fatalf("wrappedOut.Write failed: %v", err)
	}

	pb.Update(models.ProgressSnapshot{
		Discovered: 10,
		Completed:  10,
		Dirs:       2,
		Bytes:      4096,
		Done:       true,
	})
	pb.Finish()

	errText := errBuf.String()
	if !strings.Contains(errText, "100% | 10/10 items | 2 dirs | 4.0 KB") {
		t.Errorf("expected completed 100%% progress bar in stderr output, got %q", errText)
	}
	if outBuf.String() != "file.go\n" {
		t.Errorf("unexpected stdout output: %q", outBuf.String())
	}
}
