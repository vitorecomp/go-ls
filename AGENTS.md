# AGENTS.md

This file provides guidance for AI coding agents working in the `go-ls` repository.

## Project Overview

`go-ls` (`github.com/vitorecomp/go-ls`) is a client-server directory scanning, Merkle-hashing, backup comparison, and duplicate detection platform written in Go with an integrated Next.js frontend:

- **Client (`go-ls scan`, `go-ls scans`, `go-ls diff`, `go-ls duplicates`, or `cmd/client`)**: Scans local directories concurrently, prints the resumable `Scan ID` as the first line of output, computes file checksums (`crc32` or `sha256`) and folder Merkle hashes (from combined child hashes), streams entries to the server incrementally in batches (`--batch-size`, `--resume`), optionally monitors directories continuously for changes via OS filesystem events (`--live`), or formats local output (`default`, `long`, `compare` CSV, `json` NDJSON).
- **Server (`go-ls serve` or `cmd/server`)**: HTTP REST API server (`pkg/server`) that also serves the compiled Next.js static frontend (`web/out` or `/usr/local/share/go-ls/web`) on `/` (or proxies to the Next.js HMR dev server via `--dev` / `--ui-proxy`). Deployed via Podman Compose (`podman-compose.yml` + `Containerfile`) or Docker Compose (`docker-compose.yml` + `Dockerfile`) alongside PostgreSQL, with an in-memory fallback (`memory://`) for standalone use and hermetic testing.
- **Web Frontend (`web/`)**: Next.js application (`output: 'export'`) using Vanilla CSS (`web/app/globals.css`) that interacts with `/api/v1/computers`, `/api/v1/scans`, `/api/v1/scans/{id}`, `/api/v1/scans/{id}/history`, `/api/v1/diff`, `/api/v1/duplicates`, and `/healthz`.

## Repository Layout

- `main.go`: Unified CLI entrypoint supporting all subcommands (`scan` [default], `scans`, `diff`, `duplicates`, `serve`).
- `cmd/server/main.go`: Standalone HTTP server entrypoint (`go-ls-server`) used by `Containerfile` and `Dockerfile`.
- `cmd/client/main.go`: Standalone client CLI entrypoint (`go-ls-client`).
- `web/`: Next.js frontend (`web/app/page.js`, `web/app/layout.js`, `web/app/globals.css`, `web/next.config.mjs`).
- `Containerfile`, `podman-compose.yml`, & `podman-compose.dev.yml`: Multi-stage Podman build (Node 22 + Go 1.24 + Alpine), rootless Podman Compose production stack, and hot-reload development stack.
- `Dockerfile`, `docker-compose.yml`, & `docker-compose.dev.yml`: Multi-stage Docker build, Docker Compose production stack, and hot-reload development stack.
- `PRODUCT.md`, `DESIGN.md`, & `.impeccable/design.json`: Product specification and design system tokens/rules for the Next.js web console.
- `pkg/models/`: Pure domain models and option structs (zero internal imports).
  - `file.go`: `File` struct, cached UID/GID lookup, timestamps, `ComputeHash`, and `ComputeDirHash` (folder Merkle hash).
  - `system.go`: `DetectComputerID` (`/etc/machine-id` / hostname) and `DetectDriveID` (hard drive / volume UUID via `/dev/disk/by-uuid`, `/dev/disk/by-partuuid`, `/dev/disk/by-id`, `statfs.Fsid`, and `stat.Dev`).
  - `scan.go`: `Computer`, `ScanRecord`, `ScanSummary`, `ScanEntry`, `ScanHistoryEntry`, `DiffReport`, `DuplicateGroup`, `BuildScanRecord`, and `ComputeDiff`.
  - `options.go`: `LookOptions`, `OutputOptions`, `HashAlgorithm`, and `OutputFormat`.
- `pkg/looker/`: Concurrent directory walker (`looker.go`) with bounded worker semaphore, bottom-up directory Merkle hashing, and `fsnotify`-backed live filesystem watcher (`watch.go`).
- `pkg/store/`: Storage layer (`store.go`, `memory.go`, `postgres.go`) implementing `Store` (`SaveScan`, `SaveScanBatch`, `ListScans`, `ListComputers`, `GetScan`, `GetScanHistory`, `DiffScans`, `FindDuplicates`).
- `pkg/server/`: REST API and static frontend server (`server.go`) exposing `/`, `/healthz`, `/api/v1/computers`, `/api/v1/scans`, `/api/v1/scans/batch`, `/api/v1/scans/{id}`, `/api/v1/scans/{id}/history`, `/api/v1/diff`, and `/api/v1/duplicates`.
- `pkg/client/`: HTTP client (`client.go`) for communicating with `pkg/server`.
- `pkg/output/`: Local console formatters (`console.go`, `controller.go`) and client-server report formatters (`report.go`).
- `pkg/cli/`: Kong CLI structs (`cli.go`), argument normalization (`NormalizeArgs`), command execution runners (`runner.go`), and dev hot-reload supervisor (`dev.go`).
- `Makefile` & `scripts/`: Build/install automation (`Makefile`, `scripts/install.sh` for global CLI + Web UI installation) and `scripts/seed-fake-data.sh` (`scripts/seed/main.go`) to load realistic multi-computer, multi-drive, multi-revision fake data into the server.
- `documents/`: Project documentation (`README.md`, `architecture.md`, `data-model.md`, `design.md`, `todo.md`, `usage.md`, `contributing.md`).

## Build, Test, and Verification Commands

Always run these commands after making code changes:

```bash
# Build the Next.js static frontend (outputs to web/out)
cd web && npm install && npm run build && cd ..

# Format and tidy Go dependencies
gofmt -s -w .
go mod tidy

# Run static analysis
go vet ./...

# Run all unit and client-server integration tests with the race detector
go test -v -race ./...

# Build Go binaries
go build -o go-ls .
go build -o go-ls-server ./cmd/server
go build -o go-ls-client ./cmd/client
```

## Architectural & Coding Conventions

1. **Strict Dependency Direction**:
   - `pkg/models` must have **zero** dependencies on other internal packages.
   - Keep `MemoryStore` (`pkg/store/memory.go`) and `PostgresStore` (`pkg/store/postgres.go`) in sync when modifying the `store.Store` interface.
2. **Graceful Error Handling**:
   - Do **not** use `log.Fatal` or `panic` in library packages (`pkg/looker`, `pkg/models`, `pkg/output`, `pkg/store`, `pkg/server`, `pkg/client`).
3. **Deadlock-Free Concurrency**:
   - In `pkg/looker`, acquire the concurrency semaphore (`sem`) **only** around leaf I/O operations (`os.ReadDir` and `ComputeHash`), never across recursive directory walks or channel sends.
4. **Documentation Sync**:
   - When adding or modifying CLI subcommands, flags, or API endpoints, keep `README.md`, `documents/architecture.md`, `documents/data-model.md`, `documents/design.md`, `documents/todo.md`, `documents/usage.md`, and `AGENTS.md` synchronized.
