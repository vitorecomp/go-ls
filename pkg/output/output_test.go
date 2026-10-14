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
