# go-ls

A high-performance, concurrent directory listing, hashing, and comparison CLI utility written in [Go](https://go.dev/).

`go-ls` is designed to inspect large directory trees, compute file checksums in parallel, and export structured inventories (CSV or NDJSON) for backup validation and filesystem comparison.

---

## Key Features

- **Bounded Concurrent Traversal**: Walks directory trees and hashes files in parallel using a bounded worker pool so file descriptors are never exhausted on deep filesystems.
- **Multiple Output Modes**:
  - **Default**: Compact, colorized console listing.
  - **Long (`-l, --long`)**: Permissions, owner, group, byte size, modification timestamp, optional file checksum, and name.
  - **Compare (`--compare`)**: RFC 4180 CSV output (`hash,size,type,path`) designed for diffing directories and verifying backups.
  - **JSON (`--json`)**: Newline-delimited JSON (NDJSON) records for pipelines and automation tools like `jq`.
- **Configurable File & Folder Checksums**: Computes fast `crc32` (default) or cryptographic `sha256` digests (`--hash-algo`) for every file and calculates each folder's hash by combining the hashes of its contents.
- **Smart Metadata Caching**: Caches Unix UID and GID lookups in memory to eliminate redundant `/etc/passwd` and `/etc/group` reads across large scans.

---

## Installation & Running

### Run Directly (Without Installing)

From the root of the repository, use `go run .` and pass any flags or paths after it:

```bash
go run . [flags] [paths...]

# Examples:
go run . -la --hash
go run . -R --compare ./pkg
go run . --json LICENSE
```

### Build from Source

```bash
git clone https://github.com/vitorecomp/go-ls.git
cd go-ls
go build -o go-ls .
```

### Install with `go install`

```bash
go install github.com/vitorecomp/go-ls@latest
```

---

## Quick Start & Usage

```text
Usage: go-ls [<path> ...] [flags]

High-performance concurrent directory listing, hashing, and comparison tool.

Arguments:
  [<path> ...]    Files or directories to list.

Flags:
  -h, --help                Show context-sensitive help.
  -a, --all                 Show hidden files (entries starting with '.').
  -l, --long                Use a long listing format.
  -R, --recursive           List subdirectories recursively.
      --hash                Compute checksums for every file.
      --hash-algo="crc32"   Checksum algorithm to use when hashing files (crc32, sha256).
      --compare             Output CSV records (hash,size,type,path) for comparing directory trees (implies --hash).
      --json                Output newline-delimited JSON records for each entry.
      --color="auto"        Colorize the output (auto, always, never).
      --workers=0           Number of concurrent workers for directory traversal and hashing (0 uses CPU count).
      --version             Output version information and exit.
```

### Examples

```bash
# List visible entries in the current directory
go-ls

# Long listing including hidden files and CRC32 checksums
go-ls -la --hash

# Recursive long listing with SHA-256 hashes
go-ls -lR --hash --hash-algo=sha256 ./pkg

# Generate a sorted CSV manifest to compare two backup directories
go-ls -R --compare /mnt/source | sort > source.csv
go-ls -R --compare /mnt/backup | sort > backup.csv
diff -u source.csv backup.csv

# Export recursive directory metadata as NDJSON
go-ls -R --hash --json .
```

---

## Documentation

Detailed project documentation is available in the [`docs/`](./docs/) directory:

- **[Architecture Guide](./docs/architecture.md)**: Package structure, producer-consumer streaming pipeline, and concurrency model.
- **[Usage & Recipes](./docs/usage.md)**: Comprehensive CLI flag reference, output format specifications, and backup verification workflows.
- **[Contributing Guide](./docs/contributing.md)**: Development setup, testing instructions, coding conventions, and project roadmap.
- **[AI Agent Instructions (`AGENTS.md`)](./AGENTS.md)**: Context, commands, and conventions for AI coding assistants working in this repository.

---

## License

This project is licensed under the [MIT License](./LICENSE).
