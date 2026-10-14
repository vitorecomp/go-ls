# AGENTS.md

This file provides guidance for AI coding agents working in the `go-ls` repository.

## Project Overview

`go-ls` (`github.com/vitorecomp/go-ls`) is a concurrent CLI utility written in Go for listing directories, computing file checksums (`crc32` and `sha256`), and exporting structured file inventories (`default`, `long`, `compare` CSV, and `json` NDJSON) to validate backups and compare filesystems.

## Repository Layout

- `main.go`: Application entrypoint. Parses CLI flags via `github.com/alecthomas/kong`, handles `--version`, starts `looker.Run` in a background goroutine, and streams results to `output.Output`.
- `pkg/models/`: Core domain models and option structs.
  - `file.go`: `File` struct, `NewFile` constructor, cached UID/GID resolution (`FillOwnerAndGroup`), Unix timestamp extraction (`GetTimeStamps`), content hashing (`ComputeHash`), and JSON marshaling (`MarshalJSON`).
  - `options.go`: `LookOptions`, `OutputOptions`, `HashAlgorithm` (`crc32`, `sha256`), and `OutputFormat` (`default`, `long`, `compare`, `json`).
- `pkg/cli/`: Command-line interface definitions.
  - `cli.go`: `CLI` struct with Kong tags and conversion methods (`ToLookOptions()`, `ToOutputOptions()`).
  - `constants.go`: `Version` and `DefaultDateFormat` constants.
- `pkg/looker/`: Concurrent filesystem traversal.
  - `looker.go`: `Run` and `Look` functions using a bounded worker semaphore (`chan struct{}`) so deep directory scans never exhaust file descriptors.
- `pkg/output/`: Output controller and formatters.
  - `controller.go`: `Output` and `WriteOutput` functions consuming `<-chan models.File`.
  - `console.go`: `Default`, `List`, `Comparable` (RFC 4180 CSV), and `JSON` formatters.
- `docs/`: Architecture (`docs/architecture.md`), usage (`docs/usage.md`), and contributing (`docs/contributing.md`) guides.

## Build, Test, and Verification Commands

Always run these commands after making code changes:

```bash
# Format and tidy dependencies
gofmt -s -w .
go mod tidy

# Run static analysis
go vet ./...

# Run all unit tests with the race detector enabled
go test -v -race ./...

# Build the CLI binary
go build -o go-ls .
```

## Architectural & Coding Conventions

1. **Strict Dependency Direction**:
   - `pkg/models` must have **zero** dependencies on other internal packages (`pkg/cli`, `pkg/looker`, `pkg/output`).
   - Flag-to-option conversion belongs in `pkg/cli` (`ToLookOptions`, `ToOutputOptions`).
2. **Graceful Error Handling**:
   - Do **not** use `log.Fatal` or `panic` in `pkg/looker`, `pkg/models`, or `pkg/output` for per-file or per-directory filesystem errors. Return errors from `pkg/models` methods and log non-fatal traversal errors to `errOut` (`io.Writer`) in `pkg/looker`.
3. **Deadlock-Free Concurrency**:
   - In `pkg/looker`, acquire the concurrency semaphore (`sem`) **only** around leaf I/O operations (`os.ReadDir` and `ComputeHash`), never across recursive directory walks or channel sends (`out <- workingFile`).
4. **Testable I/O**:
   - Functions in `pkg/output` and `pkg/looker` accept `io.Writer` parameters (`WriteOutput`, `Run`) so unit tests can capture output using `bytes.Buffer` without mutating global `os.Stdout`/`os.Stderr`.
5. **Documentation Sync**:
   - When adding or modifying CLI flags in `pkg/cli/cli.go`, keep `README.md` and `docs/usage.md` synchronized.
