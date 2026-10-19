# Usage & Recipes Guide

This guide covers deploying the `go-ls` server and Next.js Web UI with **Podman Compose** or **Docker Compose**, uploading scans from clients, monitoring directories in real time with `--live`, inspecting directory history, comparing snapshots, finding duplicates, and running local scans.

---

## 1. Server & Web UI Deployment

### Option A: Podman Compose (`podman-compose.yml` & `podman-compose.dev.yml`)

`podman-compose.yml` (production) and `podman-compose.dev.yml` (hot-reload development) use fully-qualified `docker.io/library/...` image names and SELinux volume labels (`:Z`) for seamless rootless Podman execution:

```bash
# Production: Build the Next.js UI + Go server image and start PostgreSQL + server
podman-compose -f podman-compose.yml up -d --build
podman-compose -f podman-compose.yml down

# Development (Hot-Reload): Mounts ./ and ./web live with Go fsnotify reload + Next.js HMR
podman-compose -f podman-compose.dev.yml up
podman-compose -f podman-compose.dev.yml down
```

### Option B: Docker Compose (`docker-compose.yml` & `docker-compose.dev.yml`)

```bash
# Production: Build and start in the background
docker compose -f docker-compose.yml up -d --build
docker compose -f docker-compose.yml down

# Development (Hot-Reload): Live Go + Next.js HMR
docker compose -f docker-compose.dev.yml up
docker compose -f docker-compose.dev.yml down
```

### Accessing the Next.js Web Console

Once the server is running on port `8080`, open **`http://localhost:8080`** in your browser to use all 7 views in the left navigation rail:

1. **Operations Overview Dashboard (`dashboard`)**: Fleet-wide KPI metrics strip, per-computer volume distribution, recent snapshot revisions (`v1`, `v2`, ...), top duplicate space offenders, and a quick-start CLI guide.
2. **Directory Tree (`tree`)**: Hierarchical WinDirStat-style directory tree with subtree byte share bars, file/folder counts, Merkle digests, and an interactive file extension statistics panel.
3. **3D Cushion Map (`treemap`)**: Full-height 3D isometric hardware telemetry stage rendering directories or leaf files as extruded 3D modules with capacity LEDs, radar rings, breadcrumb navigation, and interactive zoom.
4. **File Types (`filetypes`)**: Proportional stacked extension bar and sortable file-type breakdown across the fleet or a single computer.
5. **Computers & Scans (`scans`)**: Browse tracked computers, their hardware drive UUIDs (`drive_id`), and their directory snapshots (`v1`, `v2`, ...). Toggle between **Current Files** and **Change History** for any selected directory.
6. **Snapshot Diff (`diff`)**: Select any two stored snapshots (`Base Scan` and `Target Scan`) and inspect added (`+`), modified (`~`), and deleted (`-`) entries side by side.
7. **Duplicates (`duplicates`)**: Inspect content-verified duplicate file groups across a single scan or across all stored scans, sorted by reclaimable wasted space.

---

## 2. Client-Server CLI Workflows

### Upload or Update a Directory Scan (`go-ls scan`)

When `--server` (or `GOLS_SERVER_URL`) is provided, the client detects the computer ID and hard drive UUID, scans the target path, computes file and folder Merkle hashes, and uploads the snapshot to the server. Scanning the same directory again automatically updates the snapshot on the server (`v2`, `v3`, ...) and records the diff in the directory's change history:

```bash
# Scan ./pkg recursively and store it on the server
go run . scan -R --server=http://localhost:8080 --scan-id=backup-pkg --tag=initial -q ./pkg
```

Output:

```text
Scan stored on server:
  ID:         backup-pkg (v1)
  Tag:        initial
  Computer:   workstation (8f3c1b2a...)
  Drive ID:   uuid:d47a4e2e-8140-456d-83f2-09de0cb59b8b
  Root Path:  /path/to/go-ls/pkg
  Root Hash:  4557eab2 (crc32)
  Files/Dirs: 23 files, 7 dirs (88412 bytes)
```

### Continuous Live Monitoring (`go-ls scan --live`)

Use `--live` to watch a directory for changes via OS filesystem events (`fsnotify`) and log incremental changes (`+ [ADDED]`, `~ [MODIFIED]`, `- [DELETED]`). When combined with `--server`, each detected change updates the directory snapshot on the server and appends to its change history:

```bash
# Local live monitoring
go run . -R --live ./pkg

# Live monitoring with automatic server snapshot & history updates
go run . -R --live --server=http://localhost:8080 --tag=live-watch ./pkg
```

### List & Inspect Stored Scans and History (`go-ls scans`)

```bash
# List all active directory snapshots across computers
go run . scans --server=http://localhost:8080

# Inspect a specific directory snapshot, including its revision history and current files
go run . scans backup-pkg --server=http://localhost:8080
```

### Compare Two Stored Scans (`go-ls diff`)

```bash
go run . diff scan-source scan-backup --server=http://localhost:8080
```

### Find Duplicate Files (`go-ls duplicates`)

```bash
# Search for duplicates across all stored scans
go run . duplicates --server=http://localhost:8080

# Search for duplicates within a specific scan
go run . duplicates backup-pkg --server=http://localhost:8080
```

### Seed Fake Data for Testing & UI Demos (`scripts/seed-fake-data.sh`)

To populate a running `go-ls` server with 3 realistic computers (`workstation-alpha`, `nas-storage-01`, `prod-api-node-01`), multiple hardware drive UUIDs, nested directory trees, multi-revision change histories (`v1` $\rightarrow$ `v2` $\rightarrow$ `v3`), and cross-computer duplicate files:

```bash
./scripts/seed-fake-data.sh
# or specify a custom server URL:
./scripts/seed-fake-data.sh http://localhost:8080
```

---

## 3. Standalone Server Mode (Without Containers)

You can build the Next.js frontend locally and serve it directly with `go run . serve`:

```bash
# Build the static Next.js export into ./web/out
cd web && npm install && npm run build && cd ..

# Start the Go server (automatically serves ./web/out at http://localhost:8080/)
go run . serve --listen=:8080 --db-url=memory:// --ui-dir=./web/out

# Start in hot-reload development mode (watches .go files to auto-rebuild/restart and launches Next.js HMR dev server)
go run . serve --dev --listen=:8080
```

---

## 4. Local-Only CLI Usage (No Server Required)

When `--server` is omitted, `go-ls` operates as a fast local CLI:

```bash
# Compact directory listing
go run .

# Long listing with hidden files and file + folder hashes
go run . -la --hash

# Recursive long listing with SHA-256 hashes
go run . -lR --hash --hash-algo=sha256 ./pkg

# Export RFC 4180 CSV comparison rows
go run . -R --compare ./pkg

# Export newline-delimited JSON (NDJSON)
go run . -R --hash --json ./pkg
```

---

## 5. CLI Flags & Environment Variables Reference

### Global Flag

| Flag | Default | Description |
| :--- | :--- | :--- |
| `--version` | `false` | Output version information and exit. |

### `go-ls [scan] [<path>...]` Flags

| Flag | Short | Env Var | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `--all` | `-a` | — | `false` | Show hidden files and directories (entries starting with `.`). |
| `--long` | `-l` | — | `false` | Use long listing format (permissions, owner, group, size, timestamp, path). |
| `--recursive` | `-R` | — | `false` | Traverse subdirectories recursively. |
| `--hash` | — | — | `false` | Compute checksums for every file and bottom-up Merkle hashes for every folder. |
| `--hash-algo` | — | — | `crc32` | Checksum algorithm (`crc32` or `sha256`). |
| `--compare` | — | — | `false` | Output RFC 4180 CSV records (`hash,size,type,path`) for tree comparison (implies `--hash`). |
| `--json` | — | — | `false` | Output newline-delimited JSON (NDJSON) records (or JSON summary when uploading). |
| `--color` | — | — | `auto` | Colorize console output (`auto`, `always`, `never`). |
| `--workers` | — | — | `0` | Number of concurrent traversal/hashing workers (`0` uses CPU count, minimum `4`). |
| `--live` | — | — | `false` | Continuously monitor directories via OS filesystem events (`fsnotify`; implies `--hash`). |
| `--server` | — | `GOLS_SERVER_URL` | `""` | Upload completed scan snapshots to a `go-ls` server URL (e.g. `http://localhost:8080`). |
| `--scan-id` | — | — | `""` | Optional custom snapshot ID for the uploaded scan (auto-generated if omitted). |
| `--tag` | — | `GOLS_TAG` | `""` | Optional label/tag attached to the uploaded snapshot revision. |
| `--quiet` | `-q` | — | `false` | Suppress local file listing output when uploading to `--server`. |

### `go-ls scans`, `go-ls diff`, and `go-ls duplicates` Flags

| Subcommand | Flag | Env Var | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `scans`, `diff`, `duplicates` | `--server` | `GOLS_SERVER_URL` | `http://localhost:8080` | Target `go-ls` server URL. |
| `scans`, `diff`, `duplicates` | `--json` | — | `false` | Output server response as formatted JSON. |
| `diff` | `--color` | — | `auto` | Colorize diff output (`auto`, `always`, `never`). |

### `go-ls serve` (`go-ls-server`) Flags

| Flag | Env Var | Default | Description |
| :--- | :--- | :--- | :--- |
| `--listen` | `GOLS_LISTEN_ADDR` / `GOLS_LISTEN` | `:8080` | TCP address for the HTTP REST API and Web UI server to listen on. |
| `--db-url` | `GOLS_DB_URL` | `memory://` | Storage backend connection URL (`postgres://...` or `memory://`). |
| `--ui-dir` | `GOLS_UI_DIR` / `GOLS_WEB_DIR` | `./web/out` | Directory containing the compiled Next.js static export bundle. |
| `--ui-proxy` | `GOLS_UI_PROXY` | `""` | Optional Next.js dev server URL (e.g. `http://localhost:3000`) to reverse-proxy UI & HMR requests. |
| `--dev` | `GOLS_DEV` | `false` | Enable hot-reload development supervisor (auto-rebuilds Go server on `.go` edits and proxies Next.js HMR). |

### Environment Variables Summary (`GOLS_*`)

| Environment Variable | Component | Description |
| :--- | :--- | :--- |
| `GOLS_SERVER_URL` | Client (`scan`, `scans`, `diff`, `duplicates`) | Default `go-ls` server URL (e.g. `http://localhost:8080`). |
| `GOLS_TAG` | Client (`scan`) | Default snapshot tag/label when uploading scans. |
| `GOLS_COMPUTER_ID` | Client (`pkg/models`) | Optional override for the detected host machine identifier (`/etc/machine-id`). |
| `GOLS_HOSTNAME` | Client (`pkg/models`) | Optional override for the reported OS hostname. |
| `GOLS_DRIVE_ID` | Client (`pkg/models`) | Optional override for the detected hardware volume/drive UUID (`DetectDriveID`). |
| `GOLS_LISTEN_ADDR` (`GOLS_LISTEN`) | Server (`serve`) | HTTP listen address (default `:8080`). |
| `GOLS_DB_URL` | Server (`serve`) | PostgreSQL connection string (`postgres://...`) or `memory://`. |
| `GOLS_UI_DIR` (`GOLS_WEB_DIR`) | Server (`serve`) | Filesystem path to the static Next.js build (`./web/out` or `/usr/local/share/go-ls/web`). |
| `GOLS_UI_PROXY` | Server (`serve`) | Upstream Next.js HMR dev server URL for reverse proxying. |
| `GOLS_DEV` | Server (`serve`) | Enables `--dev` hot-reload supervisor mode (`true` / `1`). |
