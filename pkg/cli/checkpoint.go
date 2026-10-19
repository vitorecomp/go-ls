package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/vitorecomp/go-ls/pkg/models"
)

type localScanCheckpoint struct {
	ID         string                      `json:"id"`
	ServerURL  string                      `json:"server_url,omitempty"`
	RootPath   string                      `json:"root_path"`
	Recursive  bool                        `json:"recursive"`
	ShowHidden bool                        `json:"show_hidden"`
	HashAlgo   string                      `json:"hash_algo"`
	Tag        string                      `json:"tag,omitempty"`
	Completed  bool                        `json:"completed"`
	Entries    map[string]models.ScanEntry `json:"entries"`
}

func checkpointFilePath(scanID string) string {
	safeID := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, strings.TrimSpace(scanID))
	if safeID == "" {
		return ""
	}
	baseDir, err := os.UserCacheDir()
	if err != nil || baseDir == "" {
		baseDir = os.TempDir()
	}
	return filepath.Join(baseDir, "go-ls", "scans", safeID+".json")
}

func loadLocalCheckpoint(scanID string) (*localScanCheckpoint, error) {
	p := checkpointFilePath(scanID)
	if p == "" {
		return nil, os.ErrNotExist
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var cp localScanCheckpoint
	if err := json.Unmarshal(raw, &cp); err != nil {
		return nil, err
	}
	if cp.Entries == nil {
		cp.Entries = make(map[string]models.ScanEntry)
	}
	return &cp, nil
}

func saveLocalCheckpoint(cp *localScanCheckpoint) {
	if cp == nil || cp.ID == "" {
		return
	}
	p := checkpointFilePath(cp.ID)
	if p == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	raw, err := json.Marshal(cp)
	if err != nil {
		return
	}
	tmpPath := p + ".tmp"
	if err := os.WriteFile(tmpPath, raw, 0o600); err != nil {
		return
	}
	_ = os.Rename(tmpPath, p)
}
