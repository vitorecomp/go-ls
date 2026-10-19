# Contributing Guide

Thank you for contributing to `go-ls`! This guide explains how to set up your development environment, build the Next.js frontend and Go backend, and run tests.

---

## Prerequisites

- **Go**: Version `1.24` or newer.
- **Node.js & npm**: Node.js `v20+` (for building the Next.js frontend in `web/`).
- **Podman Compose or Docker Compose** *(optional for unit tests, used for containerized stack deployment)*.

---

## Development Setup & Verification

### 1. Hot-Reload Development Mode (`--dev`)

Run both the Go server (with automatic `fsnotify` recompile & restart on `.go` file edits) and the Next.js Fast Refresh (HMR) dev server behind a single port (`http://localhost:8080`):

```bash
go run . serve --dev --listen=:8080
```

### 2. Build the Next.js Static Production Bundle

The Next.js app in `web/` is configured with `output: 'export'` (`web/next.config.mjs`), producing a static bundle in `web/out` that `go-ls-server` serves on `/`:

```bash
cd web
npm install
npm run build
cd ..
```

### 3. Format, Vet, and Run All Go Tests

All unit, `fsnotify` live-watcher, and HTTP client-server integration tests run hermetically using `MemoryStore` and `net/http/httptest`:

```bash
gofmt -s -w .
go mod tidy
go vet ./...
go test -v -race ./...
```

### 4. Build or Install Go Binaries Globally

```bash
# Build local binaries in the repository root
go build -o go-ls .
go build -o go-ls-server ./cmd/server
go build -o go-ls-client ./cmd/client

# Or install binaries and Web UI assets globally
make install
```

### 5. Test with Podman Compose or Docker Compose

```bash
# With Podman Compose
podman-compose -f podman-compose.yml up -d --build
go run . scan -R --server=http://localhost:8080 --scan-id=test-scan -q .
podman-compose -f podman-compose.yml down

# Or with Docker Compose
docker compose up -d --build
go run . scan -R --server=http://localhost:8080 --scan-id=test-scan -q .
docker compose down
```

---

## Coding Standards

1. **File Naming**: Use lowercase `snake_case.go` for all Go files.
2. **Package Boundaries**:
   - `pkg/models` must have zero imports from other internal packages.
   - `pkg/store` defines the `Store` interface (`SaveScan`, `ListScans`, `ListComputers`, `GetScan`, `GetScanHistory`, `DiffScans`, `FindDuplicates`); keep `MemoryStore` and `PostgresStore` behaviorally identical.
3. **Frontend Styling**:
   - The Next.js frontend (`web/`) uses semantic HTML and Vanilla CSS (`web/app/globals.css`).
4. **Concurrency Safety**:
   - Bound filesystem I/O in `pkg/looker` with the worker semaphore (`sem`), holding tokens only during leaf I/O calls (`os.ReadDir` and `ComputeHash`).
   - Always verify changes with `go test -race ./...`.
5. **Documentation Sync**:
   - Keep `README.md`, `AGENTS.md`, and the markdown files in `documents/` synchronized when adding CLI flags, models, or API routes.
