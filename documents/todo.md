# Project Roadmap & TODO (`go-ls`)

This document tracks completed engineering milestones and the prioritized backlog of upcoming enhancements for `go-ls`.

---

## Completed Milestones

- [x] **Concurrent Directory Walker & Merkle Hashing (`pkg/looker`, `pkg/models`)**
  - Bounded worker semaphore around leaf I/O (`os.ReadDir`, `ComputeHash`).
  - Support for `crc32` and `sha256` file and bottom-up directory Merkle hashing.
  - Multiple local output formats: compact default, `-l/--long`, `--compare` (RFC 4180 CSV), and `--json` (NDJSON).
- [x] **Client-Server Architecture & Storage (`pkg/server`, `pkg/store`, `pkg/client`)**
  - REST API endpoints (`/healthz`, `/api/v1/computers`, `/api/v1/scans`, `/api/v1/scans/{id}`, `/api/v1/scans/{id}/history`, `/api/v1/diff`, `/api/v1/duplicates`).
  - Dual storage backends: PostgreSQL (`PostgresStore`) and thread-safe in-memory (`MemoryStore`).
- [x] **Computer & Hard Drive Identification (`pkg/models/system.go`)**
  - Unprivileged detection of `ComputerID` (`/etc/machine-id` / hostname) and `DriveID` (`/dev/disk/by-uuid`, `/dev/disk/by-partuuid`, `/dev/disk/by-id`, `statfs.Fsid`, `stat.Dev`).
- [x] **In-Place Directory Snapshot Updates & Revision History**
  - Grouping of snapshots by **Computer** and **Drive ID**.
  - Automatic in-place snapshot versioning (`v1`, `v2`, ...) when re-scanning the same directory on the same computer/drive.
  - Full per-directory change history (`ScanHistoryEntry`) recording added, modified, and deleted entries across revisions.
- [x] **Live Filesystem Monitoring (`--live`)**
  - `fsnotify`-backed recursive directory watcher with event debouncing (`pkg/looker/watch.go`).
  - Incremental terminal change logging and automatic server snapshot updates when changes occur.
- [x] **Embedded Next.js Web Console & Container Stack (`web/`, `Containerfile`, `Dockerfile`)**
  - 7-view static Next.js UI (`Operations Overview Dashboard`, `Directory Tree`, `3D Cushion Map`, `File Types`, `Computers & Scans`, `Snapshot Diff`, `Duplicates`) served directly by the Go server on `/`.
  - Rootless Podman Compose (`podman-compose.yml` / `podman-compose.dev.yml`) and Docker Compose (`docker-compose.yml` / `docker-compose.dev.yml`) production and hot-reload dev stacks with PostgreSQL 16.

---

## Upcoming Backlog / TODO

### 1. Filtering, Ignore Rules & Scan Scope
- [ ] **`.gitignore` / `.golsignore` Support (`--ignore-file` / `--exclude`)**: Allow skipping build artifacts (`node_modules/`, `.git/`, `target/`, `vendor/`) during recursive scans and `--live` watching.
- [ ] **Max Depth Flag (`--max-depth N`)**: Limit recursive traversal depth while still optionally computing full or bounded directory hashes.
- [ ] **Minimum File Size Filter for Duplicates (`--min-size`)**: Filter out tiny files (e.g. `< 1 KB`) when running `go-ls duplicates`.

### 2. Live Monitoring & Server Real-Time Push
- [ ] **Configurable Debounce Flag (`--debounce 250ms`)**: Expose the `fsnotify` debounce interval as a CLI flag on `go-ls scan --live`.
- [ ] **Server-Sent Events (SSE) / WebSocket Stream (`GET /api/v1/events`)**: Push live directory updates from `go-ls-server` directly to open Next.js Web Console tabs without manual refresh.
- [ ] **Historical Revision Diffing (`GET /api/v1/scans/{id}/diff?from=1&to=3`)**: Allow comparing any two non-adjacent historical revisions (`vM` vs `vN`) of a single directory in the Web UI and CLI.

### 3. Security, Operations & Scale
- [ ] **API Token Authentication (`--api-key` / `GOLS_API_KEY`)**: Optional Bearer token authentication for client uploads and administrative API routes.
- [ ] **Snapshot Retention & Pruning (`DELETE /api/v1/scans/{id}`)**: Ability to delete retired directories or prune history revisions older than $N$ days.
- [ ] **Incremental Hash Caching (`mtime + size + inode`)**: Skip re-reading unchanged large files during `--live` or repeated scans when `(mtime, size, inode)` match the previous snapshot.
