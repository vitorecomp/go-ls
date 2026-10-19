# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Software engineers, homelab operators, and systems administrators auditing storage consumption, comparing directory backups, verifying file integrity, and tracking filesystem changes across multiple Linux workstations, NAS appliances, and build hosts.

## Product Purpose

`go-ls` makes multi-host, multi-drive directory state observable, verifiable, and comparable over time. Instead of treating `ls`, checksum tools, disk usage analyzers (`WinDirStat`), and backup diffing as disconnected ad-hoc utilities, `go-ls` unifies concurrent local scanning, bottom-up Merkle tree hashing, continuous filesystem event watching (`--live`), and centralized visual inspection into a single client-server platform. Success means an operator can immediately see whether two directory trees (or two points in time of the same directory) are byte-for-byte identical via a single root Merkle hash, drill down to the exact subfolder or file responsible for storage growth or drift, and reclaim wasted duplicate space across hosts.

## Positioning

Unlike standard `ls` (which provides no directory-level integrity digest) or standalone disk analyzers (which only inspect one local mount at a single point in time), `go-ls` combines:

1. **Bottom-Up Directory Merkle Hashing (`crc32` / `sha256`)**: Every folder's hash is deterministically derived from its sorted children's hashes, surfacing any deep subtree modification at the root.
2. **Unprivileged Hardware Identity**: Automatically binds every scanned directory to its physical/logical host (`ComputerID` via `/etc/machine-id`) and volume (`DriveID` via `/dev/disk/by-uuid`, `/dev/disk/by-partuuid`, `statfs.Fsid`, or `stat.Dev`) without requiring `root` or `sudo`.
3. **In-Place Snapshot Revision History (`v1`, `v2`, ...)**: Re-scanning or `--live` watching a directory updates its snapshot in place while recording compact revision deltas (`Added`, `Modified`, `Deleted`) and powering both a **WinDirStat Directory Tree** and a **3D Isometric Cushion Map**.

## Operating Context

- **CLI + Web Console Dual Workflow**: Operators run `go-ls scan -R --server=...` or `go-ls scan -R --live --server=...` from terminal sessions or cron/CI runners, then inspect fleet storage, 3D subtree modules, revision history, diffs, and duplicate groups in the browser console served on `/`.
- **Deployment Environments**: Runs either as a standalone single binary with an in-memory store (`memory://`) for instant local inspection, or as a containerized stack (`podman-compose.yml` / `docker-compose.yml`) backed by PostgreSQL 16 (`PostgresStore`), with hot-reload development stacks (`podman-compose.dev.yml` / `docker-compose.dev.yml`).

## Capabilities and Constraints

- **Core Capabilities**:
  - Concurrent bounded-worker directory traversal (`pkg/looker`) with deadlock-free leaf I/O semaphore acquisition.
  - Local CLI output modes: compact default, `-l/--long`, `--compare` (RFC 4180 CSV), and `--json` (NDJSON).
  - REST API (`pkg/server`): `/healthz`, `/api/v1/computers`, `/api/v1/scans`, `/api/v1/scans/{id}`, `/api/v1/scans/{id}/history`, `/api/v1/diff`, and `/api/v1/duplicates`.
  - Web Console (`web/`): **Operations Overview (Dashboard)**, **Directory Tree (WinDirStat)**, **3D Cushion Map**, **File Types**, **Computers & Scans**, **Snapshot Diff**, and **Duplicates**.
- **Technical Constraints**:
  - Single-binary production deployment: the Next.js frontend must compile via static export (`output: 'export'` to `web/out`) using Vanilla CSS (`web/app/globals.css`) so `go-ls-server` can serve it directly with zero external Node.js runtime required in production.
  - Strict dependency direction: `pkg/models` has zero internal package imports; `MemoryStore` and `PostgresStore` must maintain behavioral parity.
  - Unprivileged execution: client commands must never require elevated OS privileges.

## Brand Commitments

Future UI work must preserve the three visual reference commitments in `documents/references/`:

1. **Fixed Dark Sidebar & Framed Viewport (`documents/references/side-bar.jpg`)**:
   - Fixed left dark navigation rail (`#primary-sidebar`) with directional double-chevron expand/collapse arrows (`»` / `«`), top and bottom `8px` dark frame borders (no right border), a `Sync Server` action in `.sidebar-bottom`, and a green/red circular API online/offline status indicator with a white border (`#server-health-badge`).
   - Overview KPI metric cards (`.metrics-strip`) appear exclusively on the **Dashboard** tab.
2. **3D Isometric Hardware Cushion Map (`documents/references/cushion-map.jpg`)**:
   - Extruded 3D isometric hardware modules on a technical shopfloor grid inside `#treemap-workspace` (fitting `100%` of the remaining vertical screen height) with hatched floor shadow pads, corner rivets, status LEDs, isometric wall typography (`MODULE #1`), cooling vents (`//////`), 5-dot capacity indicators, active radar target rings, and a circular dark action button (`→`).
3. **Split Operations Drawer & Unified Top/Bottom Option Bars (`documents/references/option-bar.jpg`)**:
   - Card-style top header bar in normal document flow (`.viewport-header`, `margin: 1rem 1.4rem 0 1.4rem`) containing the active view title, global search pill, contextual segmented option bar (`.op-top-option-bar`), and drawer toggle icon.
   - Left collapsible "Reported Operations" card drawer (`.op-sidebar-panel`) with `.op-card-icon-badge` (`⚡` / `▣` glyphs) paired with the right main stage (`.op-main-stage`), a bottom-center floating segmented host bar (`.op-floating-bottom-bar`), a bottom-left selection/breadcrumb pill, and bottom-right `+`/`−` zoom controls.

## Evidence on Hand

- Product and architecture documentation in `README.md`, `AGENTS.md`, `documents/architecture.md`, `documents/data-model.md`, `documents/design.md`, `documents/usage.md`, and `documents/todo.md`.
- Visual design reference images in `documents/references/side-bar.jpg`, `documents/references/cushion-map.jpg`, and `documents/references/option-bar.jpg`.
- Multi-host, multi-drive, multi-revision seed generator in `scripts/seed/main.go` (`scripts/seed-fake-data.sh`).
- No external customer testimonials, pricing tiers, or marketing claims exist; future work must not fabricate them.

## Product Principles

1. **Cryptographic & Structural Truth Over Approximation**: Every byte count, percentage bar, revision diff, and 3D module dimension is derived directly from bottom-up Merkle hashes and verified scan entries.
2. **Hardware-Grounded Context**: Always anchor paths to their originating **Computer** (`hostname` / `machine-id`) and **Drive** (`uuid`), eliminating ambiguity across multi-host fleets.
3. **High-Density Operational Clarity**: Favor scannable operational cards, monospace technical readouts (hashes, sizes, versions), and direct drill-down manipulation over decorative filler.
4. **Zero-Friction Deployment & Execution**: Maintain unprivileged single-command CLI ergonomics and a self-contained static web bundle served directly by the Go binary.
