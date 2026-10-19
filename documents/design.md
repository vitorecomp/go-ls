# System Design & Engineering Decisions

This document explains the core architectural and engineering decisions behind `go-ls`.

---

## 1. Bottom-Up Directory Merkle Hashing (`pkg/looker`, `pkg/models`)

### Problem
Checking whether two directory trees (or two points in time of the same directory) are identical by comparing every file individually requires $O(N)$ comparisons. Standard `ls` only reports metadata for regular files and gives no content integrity signal for directories.

### Design
- Every regular file's content is hashed using either **IEEE CRC-32** (`crc32`, fast default) or **SHA-256** (`sha256`, cryptographic digest) in `models.File.ComputeHash`.
- Every directory's hash is computed bottom-up as a **Merkle node** (`models.File.ComputeDirHash`):
  1. `os.ReadDir` returns directory entries sorted deterministically by filename.
  2. Child files and subdirectories are hashed concurrently across worker goroutines, while preserving their sorted index position (`items[idx].Hash`).
  3. Once all children of a directory complete, their hex hashes are concatenated in deterministic sorted order into the parent directory's hasher to produce the directory's Merkle hash.
- **Result**: Even when `--recursive` (`-R`) is **not** set on the CLI output, enabling `--hash` still walks subdirectories internally so each top-level folder displays an accurate Merkle digest of its entire subtree. If a single byte changes 10 levels deep, the top-level directory's `RootHash` changes deterministically.

---

## 2. Unprivileged Hard Drive & Computer Identification (`pkg/models/system.go`)

### Problem
When multiple machines or external/internal drives are scanned and uploaded to a central `go-ls` server, paths like `/mnt/backup` or `/home/user/data` are ambiguous without knowing **which computer** and **which physical/logical drive** holds the directory. At the same time, the client CLI should work as a normal unprivileged user without requiring `sudo` or `blkid`.

### Design
1. **Computer Identifier (`DetectComputerID`)**:
   - Reads `/etc/machine-id` (or `/var/lib/dbus/machine-id`), which is world-readable (`0444`) on Linux systemd/D-Bus installations and uniquely identifies the OS installation across reboots and hostname changes.
   - Falls back to `os.Hostname()` on non-Linux systems or minimal containers.
2. **Hard Drive / Volume Identifier (`DetectDriveID`)**:
   - Calls `os.Stat(absPath)` on the target directory to inspect `syscall.Stat_t.Dev` (the kernel `major:minor` block device ID of the mounted filesystem).
   - Iterates through `/dev/disk/by-uuid` (and `/dev/disk/by-partuuid`, `/dev/disk/by-id`) using `os.Stat` (which follows the symlink to `/dev/sdaX`, `/dev/nvme0n1pX`, or `/dev/dm-X`) and compares `devStat.Rdev == dirStat.Dev`. Because `/dev/disk/by-uuid/*` symlinks and device nodes are world-stat-able (`stat()`, not `open()`), this resolves the true **Filesystem UUID** (`uuid:<uuid>`) without root privileges.
   - For virtual or network filesystems (`tmpfs`, NFS, container overlayfs) that do not appear in `/dev/disk/by-uuid`, falls back to the 64-bit OS filesystem ID from `syscall.Statfs` (`fsid:<hi>-<lo>`) and finally the kernel device number (`dev:<major>:<minor>`).

---

## 3. Computer-Contained Snapshots & In-Place Directory History (`pkg/store`)

### Problem
If every scan of `/srv/data` on `workstation-01` created an unrelated top-level scan record, the server would accumulate redundant full copies of the same directory tree and lose the explicit relationship that these scans represent successive states of the **same directory** on the **same computer and drive**.

### Design
- **Identity Key**: A tracked directory on the server is uniquely identified by `(ComputerID, DriveID, RootPath)` (or an explicit `ScanID`).
- **In-Place Snapshot Update (`store.MergeScanUpdate`)**:
  - When a directory is scanned for the first time, it is created at `Version = 1` with an initial `ScanHistoryEntry` (`v1`).
  - When the same `(ComputerID, DriveID, RootPath)` is scanned again (manually or via `--live`), the server computes `ComputeDiff(existing, incoming)`.
  - The directory's current `Entries` and summary metadata (`RootHash`, `TotalFiles`, `TotalDirs`, `TotalBytes`, `UpdatedAt`) are updated in place on the existing snapshot ID.
  - Whenever `!diff.Identical`, `Version` increments (`v2`, `v3`, ...) and a compact `ScanHistoryEntry` containing the delta (`Added`, `Deleted`, `Modified`, `PreviousHash -> RootHash`, and `Changes []DiffItem`) is prepended to `History`.
- **Benefit**: Current-state queries (`GET /api/v1/computers`, `GET /api/v1/scans`, `GET /api/v1/duplicates`) always reflect the latest state of each directory without duplicate stale entries, while `GET /api/v1/scans/{id}` and `GET /api/v1/scans/{id}/history` provide the complete audit trail of how the directory evolved over time.

---

## 4. Deadlock-Free Bounded Concurrency (`pkg/looker`)

### Problem
Recursive directory traversal where parent goroutines wait on child goroutines (`childWg.Wait()`) will deadlock if a parent holds a concurrency semaphore token while spawning and waiting for children that also need a token from the same semaphore.

### Design
- In `pkg/looker/looker.go`, the worker semaphore (`sem chan struct{}`) is acquired **strictly** around leaf I/O operations (`withSemaphore` wrapping `os.ReadDir` and `f.ComputeHash`), and released immediately before recursing into subdirectories or sending on output channels.
- This guarantees bounded file descriptor and CPU usage (`--workers`) with zero possibility of recursive semaphore starvation.

---

## 5. Event-Driven Live Monitoring with Debouncing (`pkg/looker/watch.go`)

### Problem
Editors and build tools frequently emit bursts of low-level filesystem events (`CREATE`, `WRITE`, `CHMOD`, atomic temp-file `RENAME`) within a few milliseconds. Triggering a full Merkle re-scan and server upload on every raw `fsnotify` event would cause excessive I/O and partial-write snapshots.

### Design
- `looker.Watch` registers `fsnotify` watches across the target directory tree (and dynamically registers new subdirectories on `fsnotify.Create`).
- Incoming events reset a quiet-window debounce timer (`DefaultDebounceInterval = 150ms`).
- Only after filesystem activity settles does `Watch` run a concurrent Merkle scan (`looker.Collect`), compare it against the previous baseline via `models.ComputeDiff`, and emit a `models.LiveEvent` (and server upload) if and only if `!diff.Identical`.

---

## 6. Web Console Architecture: Framed Sidebar, 7-View Telemetry Deck & 3D Isometric Cushion Map (`web/`)

### Problem
Users managing multiple computers and drives need both a high-level fleet overview and deep spatial and tabular ways to understand where disk space is being consumed, how file types are distributed, and how directory trees evolve across snapshots.

### Design
1. **Expandable Dark Sidebar Rail (`#primary-sidebar` / `.sidebar-rail`)**:
   - Frames the left edge of the viewport inside `.app-shell` (`8px` top and bottom dark borders, flush right edge) with a vertical dark rail (`64px` collapsed, `248px` expanded).
   - Houses primary navigation for all 7 views:
     1. **Operations Overview Dashboard (`dashboard`)**
     2. **Directory Tree (`tree`)**
     3. **3D Cushion Map (`treemap`)**
     4. **File Types (`filetypes`)**
     5. **Computers & Scans (`scans`)**
     6. **Snapshot Diff (`diff`)**
     7. **Duplicates (`duplicates`)**
   - Includes a bottom utility stack (`.sidebar-bottom`) containing **Sync Server** (`#sync-server-btn`), **Collapse Sidebar** (`#sidebar-toggle-btn` with directional double-chevron `»` / `«` arrows), the **Indexed Volume** storage counter, and the white-bordered **API Online/Offline** status orb (`#server-health-badge` / `.status-orb`).
2. **Card-Style Top Header (`.viewport-header`) & Split Operations Drawer (`.op-workspace-split`)**:
   - `.viewport-header` renders as a rounded card in normal document flow (`margin: 1rem 1.4rem 0 1.4rem`, `border-radius: 14px`) containing the active view title, global search pill (`#global-search-input`), and contextual segmented option bar (`.op-top-option-bar`).
   - Operational views pair a collapsible left card drawer (`.op-sidebar-panel` displaying `.op-card` telemetry cards with `.op-card-icon-badge` `⚡` / `▣` glyphs, circular volume share rings, and host-to-drive route connectors) with the right main stage (`.op-main-stage`) and a floating bottom host floor selector (`.op-floating-bottom-bar`).
3. **Operations Overview Dashboard (`activeTab === 'dashboard'`)**:
   - Exclusively renders the 5-card KPI overview strip (`.metrics-strip`), alongside fleet storage share bars, drive UUID chips, recent snapshot revisions (`v1`, `v2`, ...), top duplicate space offenders, and an interactive CLI onboarding checklist.
4. **WinDirStat Directory Tree (`activeTab === 'tree'`)**:
   - Reconstructs the full hierarchical tree (`Computer` -> `Tracked Root Directory` -> `Subfolders` -> `Files`) from each computer's stored `ScanRecord` entries (`buildWinDirTree`).
   - Computes bottom-up recursive `subtreeBytes`, `fileCount`, `dirCount`, `pctOfParent` (rendered with WinDirStat-style horizontal progress bars), and `pctOfRoot`, paired with an interactive **Extension Statistics List**.
5. **3D Isometric Cushion Map (`activeTab === 'treemap'`)**:
   - `#treemap-workspace` fits the remaining vertical viewport height (`height: calc(100vh - 116px)`, `min-height: 500px`) with `.iso-cushion-stage` (`height: 100%`) so the entire 3D stage is visible without vertical scrolling.
   - Renders squarified treemap subdivisions (`computeTreemapLayout`) as extruded 3D anodized-aluminum hardware modules on a `30°` isometric shopfloor grid (`cos(30°) = 0.8660254`, `sin(30°) = 0.5`) with hatched floor shadow pads, corner rivets, status LEDs, isometric wall labels, cooling vents (`//////`), 5-dot capacity indicators, active radar target rings, and floating HUD controls (legend toggle, breadcrumb bar, host floor bar, and `+`/`−` zoom dock).
6. **File Types, Computers & Scans, Snapshot Diff, and Duplicates (`filetypes`, `scans`, `diff`, `duplicates`)**:
   - **File Types (`filetypes`)**: Stacked proportional distribution bar and interactive extension breakdown table across computers or selected snapshots.
   - **Computers & Scans (`scans`)**: Full file/directory inventory table (**Current Files**) and timeline audit log (**Change History**) with Merkle hash verification.
   - **Snapshot Diff (`diff`)**: Side-by-side comparison of any two stored snapshots highlighting `+Added`, `~Modified`, and `-Deleted` entries.
   - **Duplicates (`duplicates`)**: Content-addressed duplicate file groups (`(hash, size)`) sorted by reclaimable wasted bytes across a single scan or the entire fleet.
