# Contributing Guide

Thank you for your interest in contributing to `go-ls`! This guide explains how to set up your environment, run tests, and submit changes.

---

## Prerequisites

- **Go**: Version `1.24` or newer.
- **Operating System**: Linux / Unix environment (uses `syscall.Stat_t` for extended file timestamps and UID/GID resolution).

---

## Development Workflow

### 1. Clone and Build

```bash
git clone https://github.com/vitorecomp/go-ls.git
cd go-ls
go mod tidy
go build -o go-ls .
```

### 2. Run Static Analysis and Tests

Always run `go vet` and the unit test suite with the race detector enabled before submitting changes:

```bash
go vet ./...
go test -v -race ./...
```

---

## Coding Standards & Best Practices

1. **File Naming**: Use lowercase `snake_case.go` for all Go source and test files (e.g., `file.go`, `file_test.go`).
2. **Package Boundaries**:
   - Keep `pkg/models` free of imports from `pkg/cli`, `pkg/looker`, or `pkg/output`.
   - Avoid package-level mutable state for CLI arguments; pass `models.LookOptions` and `models.OutputOptions` explicitly.
3. **Error Handling**:
   - Never call `log.Fatal` or `panic` inside library packages (`pkg/models`, `pkg/looker`, `pkg/output`) when encountering unreadable files or permission errors during traversal. Return an `error` or write a diagnostic message to `errOut` so the rest of the scan can complete.
4. **Concurrency Safety**:
   - Bound concurrent file I/O using the worker semaphore in `pkg/looker`.
   - Ensure any shared caches (such as UID/GID lookups) are safe for concurrent access (`sync.Map` or `sync.RWMutex`) and verified with `go test -race ./...`.
5. **Documentation**:
   - Document all exported types, functions, methods, and constants with standard Go doc comments.
   - Update `README.md` and `docs/usage.md` whenever CLI flags or output formats change.

---

## Roadmap

Planned enhancements for future releases:

- **Database Output Backend**: Optional SQLite and PostgreSQL persistence (`pkg/storage`) for storing historical directory snapshots and incremental comparisons across runs.
- **Cross-Platform `Stat` Abstraction**: Build-tagged platform helpers (`file_unix.go`, `file_windows.go`) for native Windows support.
- **Custom Go Templates**: User-supplied `--template` formatting for bespoke console output.
- **Directory Merkle Hashes**: Bottom-up directory tree hashing computed from child file checksums.
