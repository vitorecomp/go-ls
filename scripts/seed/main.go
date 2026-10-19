// Command seed populates a running go-ls server with realistic multi-computer,
// multi-drive, multi-revision directory snapshots for testing the Dashboard,
// WinDirStat Directory Tree & Cushion Treemap, Snapshot Diffing, and Duplicate Detection.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/vitorecomp/go-ls/pkg/client"
	"github.com/vitorecomp/go-ls/pkg/models"
)

type fakeFileSpec struct {
	relPath string
	size    int64
	hashKey string // deterministic seed for content hash (same hashKey + size = duplicate file)
	owner   string
	group   string
	mode    string
}

func shortHash(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(sum[:])[:16]
}

// buildSnapshot constructs a complete models.ScanRecord from leaf file specs,
// automatically synthesizing all intermediate directory entries and computing
// bottom-up Merkle hashes for every folder and the root directory.
func buildSnapshot(
	id string,
	tag string,
	computerID string,
	hostname string,
	driveID string,
	rootPath string,
	createdAt time.Time,
	files []fakeFileSpec,
) models.ScanRecord {
	dirSet := make(map[string]bool)
	entriesByPath := make(map[string]models.ScanEntry)
	childrenByDir := make(map[string][]string)

	addChild := func(parent, child string) {
		for _, existing := range childrenByDir[parent] {
			if existing == child {
				return
			}
		}
		childrenByDir[parent] = append(childrenByDir[parent], child)
	}

	var totalFiles, totalDirs int
	var totalBytes int64

	for _, f := range files {
		rel := strings.TrimPrefix(path.Clean(f.relPath), "/")
		if rel == "." || rel == "" {
			continue
		}

		// Ensure all parent directories exist
		parts := strings.Split(rel, "/")
		parent := "."
		for i := 0; i < len(parts)-1; i++ {
			dirRel := strings.Join(parts[:i+1], "/")
			if !dirSet[dirRel] {
				dirSet[dirRel] = true
				totalDirs++
				entriesByPath[dirRel] = models.ScanEntry{
					RelPath: dirRel,
					AbsPath: path.Join(rootPath, dirRel),
					Name:    parts[i],
					IsDir:   true,
					Size:    4096,
					Mode:    "drwxr-xr-x",
					Owner:   f.owner,
					Group:   f.group,
					ModTime: createdAt,
				}
			}
			addChild(parent, dirRel)
			parent = dirRel
		}

		addChild(parent, rel)
		fileHash := shortHash(f.hashKey)
		mode := f.mode
		if mode == "" {
			mode = "-rw-r--r--"
		}
		entriesByPath[rel] = models.ScanEntry{
			RelPath: rel,
			AbsPath: path.Join(rootPath, rel),
			Name:    path.Base(rel),
			IsDir:   false,
			Size:    f.size,
			Mode:    mode,
			Owner:   f.owner,
			Group:   f.group,
			ModTime: createdAt,
			Hash:    fileHash,
		}
		totalFiles++
		totalBytes += f.size
	}

	// Compute bottom-up Merkle hashes for every directory
	var computeMerkle func(dirRel string) string
	computeMerkle = func(dirRel string) string {
		children := childrenByDir[dirRel]
		sort.Strings(children)
		var childHashes []string
		for _, ch := range children {
			entry := entriesByPath[ch]
			if entry.IsDir {
				h := computeMerkle(ch)
				entry.Hash = h
				entriesByPath[ch] = entry
				childHashes = append(childHashes, h)
			} else {
				childHashes = append(childHashes, entry.Hash)
			}
		}
		combined := shortHash("dir:" + strings.Join(childHashes, "|"))
		return combined
	}

	rootHash := computeMerkle(".")

	allPaths := make([]string, 0, len(entriesByPath))
	for p := range entriesByPath {
		allPaths = append(allPaths, p)
	}
	sort.Strings(allPaths)

	entries := make([]models.ScanEntry, 0, len(allPaths))
	for _, p := range allPaths {
		entries = append(entries, entriesByPath[p])
	}

	return models.ScanRecord{
		ScanSummary: models.ScanSummary{
			ID:         id,
			Tag:        tag,
			ComputerID: computerID,
			Hostname:   hostname,
			DriveID:    driveID,
			RootPath:   rootPath,
			RootHash:   rootHash,
			HashAlgo:   string(models.HashSHA256),
			Version:    1,
			TotalFiles: totalFiles,
			TotalDirs:  totalDirs,
			TotalBytes: totalBytes,
			CreatedAt:  createdAt,
			UpdatedAt:  createdAt,
		},
		Entries: entries,
	}
}

func main() {
	defaultServer := os.Getenv("GOLS_SERVER_URL")
	if defaultServer == "" {
		defaultServer = "http://localhost:8080"
	}

	serverURL := flag.String("server", defaultServer, "Target go-ls server base URL")
	flag.Parse()

	cli := client.New(*serverURL)
	ctx := context.Background()

	if err := cli.Health(ctx); err != nil {
		log.Fatalf("Cannot reach go-ls server at %s: %v", *serverURL, err)
	}

	now := time.Now().UTC()
	log.Printf("Seeding fake computers, drives, directory trees, and revision histories into %s...", *serverURL)

	// =========================================================================
	// 1. Computer: workstation-alpha (Primary NVMe + External USB-C SSD Backup)
	// =========================================================================
	compAlphaID := "fedcba9876543210fedcba9876543210"
	compAlphaHost := "workstation-alpha"
	driveNVMe := "uuid:9f84-41c2-nvme0n1p2"
	driveBackupSSD := "uuid:3b12-90aa-sda1-backup"

	// Snapshot 1A (v1): /home/alice/projects/go-ls-engine (Baseline)
	engineV1Files := []fakeFileSpec{
		{relPath: "go.mod", size: 1420, hashKey: "engine-gomod-v1", owner: "alice", group: "staff"},
		{relPath: "go.sum", size: 18450, hashKey: "engine-gosum-v1", owner: "alice", group: "staff"},
		{relPath: "README.md", size: 8400, hashKey: "engine-readme-v1", owner: "alice", group: "staff"},
		{relPath: "cmd/server/main.go", size: 12400, hashKey: "engine-server-main-v1", owner: "alice", group: "staff"},
		{relPath: "cmd/client/main.go", size: 9600, hashKey: "engine-client-main-v1", owner: "alice", group: "staff"},
		{relPath: "pkg/merkle/tree.go", size: 34800, hashKey: "engine-merkle-tree-v1", owner: "alice", group: "staff"},
		{relPath: "pkg/merkle/hasher.go", size: 22100, hashKey: "engine-merkle-hasher-v1", owner: "alice", group: "staff"},
		{relPath: "pkg/store/postgres.go", size: 68400, hashKey: "engine-store-pg-v1", owner: "alice", group: "staff"},
		{relPath: "pkg/store/memory.go", size: 41200, hashKey: "engine-store-mem-v1", owner: "alice", group: "staff"},
		{relPath: "pkg/store/migrations/001_init.sql", size: 6200, hashKey: "engine-sql-001", owner: "alice", group: "staff"},
		{relPath: "web/app/page.js", size: 94500, hashKey: "engine-web-page-v1", owner: "alice", group: "staff"},
		{relPath: "web/app/globals.css", size: 48200, hashKey: "engine-web-css-v1", owner: "alice", group: "staff"},
		{relPath: "web/public/architecture-diagram.svg", size: 145000, hashKey: "shared-arch-svg", owner: "alice", group: "staff"},
		{relPath: "dist/go-ls-linux-amd64", size: 18450000, hashKey: "engine-bin-linux-v1", owner: "alice", group: "staff", mode: "-rwxr-xr-x"},
		{relPath: "dist/go-ls-darwin-arm64", size: 17820000, hashKey: "engine-bin-darwin-v1", owner: "alice", group: "staff", mode: "-rwxr-xr-x"},
		{relPath: "docs/legacy-notes.md", size: 5100, hashKey: "engine-legacy-notes", owner: "alice", group: "staff"},
	}

	// Snapshot 1A (v2): /home/alice/projects/go-ls-engine (Live update: modified code, added WinDirStat & migrations, deleted legacy-notes.md)
	engineV2Files := []fakeFileSpec{
		{relPath: "go.mod", size: 1580, hashKey: "engine-gomod-v2", owner: "alice", group: "staff"},
		{relPath: "go.sum", size: 19200, hashKey: "engine-gosum-v2", owner: "alice", group: "staff"},
		{relPath: "README.md", size: 11200, hashKey: "engine-readme-v2", owner: "alice", group: "staff"},
		{relPath: "cmd/server/main.go", size: 12400, hashKey: "engine-server-main-v1", owner: "alice", group: "staff"},
		{relPath: "cmd/client/main.go", size: 9600, hashKey: "engine-client-main-v1", owner: "alice", group: "staff"},
		{relPath: "pkg/merkle/tree.go", size: 42600, hashKey: "engine-merkle-tree-v2", owner: "alice", group: "staff"},
		{relPath: "pkg/merkle/hasher.go", size: 22100, hashKey: "engine-merkle-hasher-v1", owner: "alice", group: "staff"},
		{relPath: "pkg/merkle/watch.go", size: 28900, hashKey: "engine-merkle-watch-v2", owner: "alice", group: "staff"},
		{relPath: "pkg/store/postgres.go", size: 79400, hashKey: "engine-store-pg-v2", owner: "alice", group: "staff"},
		{relPath: "pkg/store/memory.go", size: 46800, hashKey: "engine-store-mem-v2", owner: "alice", group: "staff"},
		{relPath: "pkg/store/migrations/001_init.sql", size: 6200, hashKey: "engine-sql-001", owner: "alice", group: "staff"},
		{relPath: "pkg/store/migrations/002_scan_history.sql", size: 4800, hashKey: "engine-sql-002", owner: "alice", group: "staff"},
		{relPath: "web/app/page.js", size: 132400, hashKey: "engine-web-page-v2", owner: "alice", group: "staff"},
		{relPath: "web/app/globals.css", size: 61500, hashKey: "engine-web-css-v2", owner: "alice", group: "staff"},
		{relPath: "web/public/architecture-diagram.svg", size: 145000, hashKey: "shared-arch-svg", owner: "alice", group: "staff"},
		{relPath: "dist/go-ls-linux-amd64", size: 19640000, hashKey: "engine-bin-linux-v2", owner: "alice", group: "staff", mode: "-rwxr-xr-x"},
		{relPath: "dist/go-ls-darwin-arm64", size: 17820000, hashKey: "engine-bin-darwin-v1", owner: "alice", group: "staff", mode: "-rwxr-xr-x"},
	}

	// Snapshot 1B (v1): /mnt/external-ssd/backups/go-ls-engine-backup (Backup on second drive — matches v1 so Compare/Diff against live v2 shows exact drift!)
	backupDriveFiles := append([]fakeFileSpec{}, engineV1Files...)

	// Snapshot 1C (v1): /home/alice/media-library (Rich file extensions & sizes for WinDirStat treemap + duplicates)
	mediaLibFiles := []fakeFileSpec{
		{relPath: "videos/keynote-demo-4k.mp4", size: 84500000, hashKey: "shared-keynote-mp4", owner: "alice", group: "staff"},
		{relPath: "videos/architecture-walkthrough.mkv", size: 52400000, hashKey: "media-walkthrough-mkv", owner: "alice", group: "staff"},
		{relPath: "photos/2026-offsite/team-summit-01.jpg", size: 6840000, hashKey: "shared-photo-01", owner: "alice", group: "staff"},
		{relPath: "photos/2026-offsite/team-summit-02.jpg", size: 7120000, hashKey: "media-photo-02", owner: "alice", group: "staff"},
		{relPath: "photos/2026-offsite/whiteboard-design.png", size: 9450000, hashKey: "shared-whiteboard-png", owner: "alice", group: "staff"},
		{relPath: "music/flac/ambient-coding-session-01.flac", size: 38200000, hashKey: "media-flac-01", owner: "alice", group: "staff"},
		{relPath: "music/flac/ambient-coding-session-02.flac", size: 41600000, hashKey: "media-flac-02", owner: "alice", group: "staff"},
		{relPath: "archives/dataset-sample-2026.tar.gz", size: 29500000, hashKey: "shared-dataset-tar", owner: "alice", group: "staff"},
		{relPath: "archives/dataset-sample-2026-copy.tar.gz", size: 29500000, hashKey: "shared-dataset-tar", owner: "alice", group: "staff"},
	}

	// =========================================================================
	// 2. Computer: nas-storage-01 (ZFS RAIDZ2 Pool with AI Datasets & Shared Backups)
	// =========================================================================
	compNASID := "11223344556677889900aabbccddeeff"
	compNASHost := "nas-storage-01"
	driveZFS := "uuid:zfs-tank-raidz2-01"

	nasDatasetsV1 := []fakeFileSpec{
		{relPath: "models/embeddings-v1.bin", size: 142000000, hashKey: "nas-model-embed-v1", owner: "mlops", group: "research"},
		{relPath: "models/reranker-quantized.wasm", size: 48500000, hashKey: "nas-model-rerank", owner: "mlops", group: "research"},
		{relPath: "datasets/train-shard-0001.parquet", size: 96400000, hashKey: "nas-parquet-0001", owner: "mlops", group: "research"},
		{relPath: "datasets/train-shard-0002.parquet", size: 94800000, hashKey: "nas-parquet-0002", owner: "mlops", group: "research"},
		{relPath: "datasets/validation.parquet", size: 24600000, hashKey: "nas-parquet-val-v1", owner: "mlops", group: "research"},
		{relPath: "configs/hyperparams.yaml", size: 4200, hashKey: "nas-config-v1", owner: "mlops", group: "research"},
		{relPath: "archives/dataset-sample-2026.tar.gz", size: 29500000, hashKey: "shared-dataset-tar", owner: "mlops", group: "research"},
	}

	nasDatasetsV2 := []fakeFileSpec{
		{relPath: "models/embeddings-v2.bin", size: 156000000, hashKey: "nas-model-embed-v2", owner: "mlops", group: "research"},
		{relPath: "models/reranker-quantized.wasm", size: 48500000, hashKey: "nas-model-rerank", owner: "mlops", group: "research"},
		{relPath: "datasets/train-shard-0001.parquet", size: 96400000, hashKey: "nas-parquet-0001", owner: "mlops", group: "research"},
		{relPath: "datasets/train-shard-0002.parquet", size: 94800000, hashKey: "nas-parquet-0002", owner: "mlops", group: "research"},
		{relPath: "datasets/train-shard-0003.parquet", size: 98100000, hashKey: "nas-parquet-0003", owner: "mlops", group: "research"},
		{relPath: "datasets/validation.parquet", size: 28900000, hashKey: "nas-parquet-val-v2", owner: "mlops", group: "research"},
		{relPath: "configs/hyperparams.yaml", size: 5100, hashKey: "nas-config-v2", owner: "mlops", group: "research"},
		{relPath: "archives/dataset-sample-2026.tar.gz", size: 29500000, hashKey: "shared-dataset-tar", owner: "mlops", group: "research"},
	}

	nasSharedMedia := []fakeFileSpec{
		{relPath: "talks/keynote-demo-4k.mp4", size: 84500000, hashKey: "shared-keynote-mp4", owner: "mlops", group: "research"},
		{relPath: "talks/keynote-raw-cut-01.mp4", size: 62000000, hashKey: "nas-raw-01", owner: "mlops", group: "research"},
		{relPath: "talks/keynote-raw-cut-02.mp4", size: 58000000, hashKey: "nas-raw-02", owner: "mlops", group: "research"},
		{relPath: "talks/keynote-raw-cut-03.mp4", size: 54000000, hashKey: "nas-raw-03", owner: "mlops", group: "research"},
		{relPath: "staging/tmp-export-01.tar", size: 41000000, hashKey: "nas-tmp-01", owner: "mlops", group: "research"},
		{relPath: "staging/tmp-export-02.tar", size: 39000000, hashKey: "nas-tmp-02", owner: "mlops", group: "research"},
		{relPath: "staging/tmp-export-03.tar", size: 37000000, hashKey: "nas-tmp-03", owner: "mlops", group: "research"},
		{relPath: "staging/tmp-export-04.tar", size: 35000000, hashKey: "nas-tmp-04", owner: "mlops", group: "research"},
		{relPath: "staging/tmp-export-05.tar", size: 33000000, hashKey: "nas-tmp-05", owner: "mlops", group: "research"},
		{relPath: "staging/tmp-export-06.tar", size: 31000000, hashKey: "nas-tmp-06", owner: "mlops", group: "research"},
		{relPath: "diagrams/architecture-diagram.svg", size: 145000, hashKey: "shared-arch-svg", owner: "mlops", group: "research"},
		{relPath: "diagrams/whiteboard-design.png", size: 9450000, hashKey: "shared-whiteboard-png", owner: "mlops", group: "research"},
		{relPath: "events/team-summit-01.jpg", size: 6840000, hashKey: "shared-photo-01", owner: "mlops", group: "research"},
		{relPath: "reports/q3-storage-audit.json", size: 1840000, hashKey: "nas-audit-json", owner: "mlops", group: "research"},
	}

	nasSharedMediaPruned := []fakeFileSpec{
		{relPath: "talks/keynote-demo-4k.mp4", size: 84500000, hashKey: "shared-keynote-mp4", owner: "mlops", group: "research"},
		{relPath: "diagrams/architecture-diagram.svg", size: 145000, hashKey: "shared-arch-svg", owner: "mlops", group: "research"},
		{relPath: "diagrams/whiteboard-design.png", size: 9450000, hashKey: "shared-whiteboard-png", owner: "mlops", group: "research"},
		{relPath: "events/team-summit-01.jpg", size: 6840000, hashKey: "shared-photo-01", owner: "mlops", group: "research"},
		{relPath: "reports/q3-storage-audit.json", size: 1840000, hashKey: "nas-audit-json", owner: "mlops", group: "research"},
	}

	// =========================================================================
	// 3. Computer: prod-api-node-01 (Production Cloud Server with 3 Revisions)
	// =========================================================================
	compProdID := "a1b2c3d4e5f60718293a4b5c6d7e8f90"
	compProdHost := "prod-api-node-01"
	driveEBS := "uuid:ebs-gp3-vol-0a88bc12"

	prodReleaseV1 := []fakeFileSpec{
		{relPath: "bin/api-gateway", size: 26400000, hashKey: "prod-gateway-v1", owner: "deploy", group: "deploy", mode: "-rwxr-xr-x"},
		{relPath: "bin/worker-daemon", size: 21800000, hashKey: "prod-worker-v1", owner: "deploy", group: "deploy", mode: "-rwxr-xr-x"},
		{relPath: "config/production.yaml", size: 6400, hashKey: "prod-cfg-v1", owner: "deploy", group: "deploy"},
		{relPath: "static/bundle.js", size: 640000, hashKey: "prod-bundle-js-v1", owner: "deploy", group: "deploy"},
		{relPath: "static/styles.css", size: 92000, hashKey: "prod-styles-css-v1", owner: "deploy", group: "deploy"},
		{relPath: "logs/access.log", size: 4200000, hashKey: "prod-access-log-v1", owner: "deploy", group: "deploy"},
	}

	prodReleaseV2 := []fakeFileSpec{
		{relPath: "bin/api-gateway", size: 27100000, hashKey: "prod-gateway-v2", owner: "deploy", group: "deploy", mode: "-rwxr-xr-x"},
		{relPath: "bin/worker-daemon", size: 21800000, hashKey: "prod-worker-v1", owner: "deploy", group: "deploy", mode: "-rwxr-xr-x"},
		{relPath: "config/production.yaml", size: 6850, hashKey: "prod-cfg-v2", owner: "deploy", group: "deploy"},
		{relPath: "static/bundle.js", size: 712000, hashKey: "prod-bundle-js-v2", owner: "deploy", group: "deploy"},
		{relPath: "static/styles.css", size: 98400, hashKey: "prod-styles-css-v2", owner: "deploy", group: "deploy"},
		{relPath: "logs/access.log", size: 8900000, hashKey: "prod-access-log-v2", owner: "deploy", group: "deploy"},
	}

	prodReleaseV3 := []fakeFileSpec{
		{relPath: "bin/api-gateway", size: 27150000, hashKey: "prod-gateway-v3-hotfix", owner: "deploy", group: "deploy", mode: "-rwxr-xr-x"},
		{relPath: "bin/worker-daemon", size: 22400000, hashKey: "prod-worker-v3", owner: "deploy", group: "deploy", mode: "-rwxr-xr-x"},
		{relPath: "config/production.yaml", size: 6850, hashKey: "prod-cfg-v2", owner: "deploy", group: "deploy"},
		{relPath: "static/bundle.js", size: 712000, hashKey: "prod-bundle-js-v2", owner: "deploy", group: "deploy"},
		{relPath: "static/styles.css", size: 98400, hashKey: "prod-styles-css-v2", owner: "deploy", group: "deploy"},
		{relPath: "static/hotfix-banner.js", size: 14200, hashKey: "prod-hotfix-js-v3", owner: "deploy", group: "deploy"},
		{relPath: "logs/access.log", size: 14500000, hashKey: "prod-access-log-v3", owner: "deploy", group: "deploy"},
	}

	sequence := []models.ScanRecord{
		// Workstation Alpha: go-ls-engine v1 -> v2
		buildSnapshot("alpha-go-ls-engine", "baseline-v1", compAlphaID, compAlphaHost, driveNVMe, "/home/alice/projects/go-ls-engine", now.Add(-6*time.Hour), engineV1Files),
		buildSnapshot("alpha-go-ls-engine", "live-sync-v2", compAlphaID, compAlphaHost, driveNVMe, "/home/alice/projects/go-ls-engine", now.Add(-1*time.Hour), engineV2Files),
		// Workstation Alpha: External Backup SSD
		buildSnapshot("alpha-external-backup", "ssd-backup-v1", compAlphaID, compAlphaHost, driveBackupSSD, "/mnt/external-ssd/backups/go-ls-engine-backup", now.Add(-5*time.Hour), backupDriveFiles),
		// Workstation Alpha: Media Library
		buildSnapshot("alpha-media-library", "media-v1", compAlphaID, compAlphaHost, driveNVMe, "/home/alice/media-library", now.Add(-3*time.Hour), mediaLibFiles),

		// NAS Storage 01: AI Datasets v1 -> v2 & Shared Media (with staging retention prune)
		buildSnapshot("nas-ai-datasets", "epoch-1", compNASID, compNASHost, driveZFS, "/srv/nfs/ai-datasets", now.Add(-8*time.Hour), nasDatasetsV1),
		buildSnapshot("nas-ai-datasets", "epoch-2", compNASID, compNASHost, driveZFS, "/srv/nfs/ai-datasets", now.Add(-2*time.Hour), nasDatasetsV2),
		buildSnapshot("nas-shared-media", "team-share-v1", compNASID, compNASHost, driveZFS, "/srv/nfs/team-shared-media", now.Add(-4*time.Hour), nasSharedMedia),
		buildSnapshot("nas-shared-media", "retention-prune-v2", compNASID, compNASHost, driveZFS, "/srv/nfs/team-shared-media", now.Add(-45*time.Minute), nasSharedMediaPruned),

		// Prod API Node 01: Platform Release v1 -> v2 -> v3
		buildSnapshot("prod-platform-release", "release-2026.10.0", compProdID, compProdHost, driveEBS, "/var/www/platform-release", now.Add(-12*time.Hour), prodReleaseV1),
		buildSnapshot("prod-platform-release", "release-2026.10.1", compProdID, compProdHost, driveEBS, "/var/www/platform-release", now.Add(-4*time.Hour), prodReleaseV2),
		buildSnapshot("prod-platform-release", "hotfix-2026.10.2", compProdID, compProdHost, driveEBS, "/var/www/platform-release", now.Add(-25*time.Minute), prodReleaseV3),
	}

	for _, rec := range sequence {
		summary, err := cli.UploadScan(ctx, rec)
		if err != nil {
			log.Fatalf("Failed uploading snapshot %s (%s:%s): %v", rec.ID, rec.Hostname, rec.RootPath, err)
		}
		fmt.Printf("  ✓ Uploaded %-24s | %-18s | %-44s | v%d | %2d files | %s\n",
			summary.ID,
			summary.Hostname,
			summary.RootPath,
			summary.Version,
			summary.TotalFiles,
			summary.RootHash[:10],
		)
	}

	fmt.Printf("\nSuccessfully seeded %d snapshot revisions across 3 computers into %s!\n", len(sequence), *serverURL)
}
