# Usage & Recipes Guide

This guide covers all command-line flags, output formats, and common workflows for `go-ls`.

---

## Command-Line Reference

| Flag | Short | Default | Description |
| :--- | :---: | :---: | :--- |
| `--all` | `-a` | `false` | Include hidden files and directories (names starting with `.`). |
| `--long` | `-l` | `false` | Display detailed metadata (mode, owner, group, size, date, optional hash, name). |
| `--recursive` | `-R` | `false` | Recursively traverse subdirectories. |
| `--hash` | | `false` | Compute content checksums for all files and combined content hashes for directories. |
| `--hash-algo` | | `crc32` | Hash algorithm to use (`crc32` or `sha256`). |
| `--compare` | | `false` | Output RFC 4180 CSV rows (`hash,size,type,path`). Automatically enables `--hash`. |
| `--json` | | `false` | Output newline-delimited JSON (NDJSON) objects for each entry. |
| `--color` | | `auto` | Control ANSI color output (`auto`, `always`, `never`). |
| `--workers` | | `0` | Maximum concurrent I/O workers (`0` defaults to CPU core count, minimum 4). |
| `--version` | | `false` | Print `go-ls` version and exit. |
| `--help` | `-h` | `false` | Show help message and exit. |

---

## Output Formats

### 1. Default Format

Prints entry names inline with directories highlighted in bold blue:

```bash
go-ls
```

### 2. Long Format (`-l, --long`)

Displays file permissions, owner, group, size in bytes, modification time, optional checksum (when `--hash` is passed), and name:

```bash
go-ls -l --hash
```

Example output:

```text
-rw-r--r-- vitor    vitor        1072 Oct 14 12:24 f58f6d5a LICENSE
-rw-r--r-- vitor    vitor        2840 Oct 14 12:30 8b2874d5 README.md
drwxr-xr-x vitor    vitor        4096 Oct 14 12:30 9c1b4a02 pkg
```

### 3. Compare CSV Format (`--compare`)

Outputs RFC 4180 CSV records with four columns:

1. `hash`: File checksum or directory combined content hash
2. `size`: Size in bytes
3. `type`: `file` or `directory`
4. `path`: Absolute path (properly escaped if the path contains commas or quotes)

```bash
go-ls -R --compare ./pkg
```

Example output:

```csv
8a19d2f1,1420,file,/path/to/go-ls/pkg/cli/cli.go
4e19b0c2,4096,directory,/path/to/go-ls/pkg/cli
```

### 4. JSON Format (`--json`)

Outputs one JSON object per line (NDJSON), ideal for piping to `jq` or ingesting into log/data pipelines:

```bash
go-ls --hash --json LICENSE
```

Example output:

```json
{"name":"LICENSE","path":"/path/to/go-ls/LICENSE","type":"file","is_dir":false,"size":1072,"mode":"-rw-r-----","owner":"vitor","group":"primarygroup","mod_time":"2026-10-14T12:24:00-03:00","access_time":"2026-10-14T12:24:00-03:00","change_time":"2026-10-14T12:24:00-03:00","hash":"a83f91bc"}
```

---

## Common Workflows & Recipes

### Verify Backups or Deep Copies

Because `go-ls` walks directories concurrently, output order is non-deterministic. Pipe the output through `sort` when comparing two directory trees:

```bash
# Compare using fast CRC32 checksums
go-ls -R --compare /data/primary | sort > primary.csv
go-ls -R --compare /data/replica | sort > replica.csv
diff -u primary.csv replica.csv
```

For cryptographic verification across untrusted storage, switch to `sha256`:

```bash
go-ls -R --compare --hash-algo=sha256 /data/primary | sort > primary.sha256.csv
```

### Find the Largest Files Using `--json` and `jq`

```bash
go-ls -R --json . | jq -s 'map(select(.is_dir == false)) | sort_by(-.size) | .[:10]'
```
