package looker

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/vitorecomp/go-ls/pkg/models"
)

// DefaultDebounceInterval is the quiet window used to coalesce bursts of filesystem events.
const DefaultDebounceInterval = 150 * time.Millisecond

// WatchCallback is invoked whenever a debounced filesystem event produces a non-empty diff
// against the previous scan snapshot.
type WatchCallback func(files []models.File, prevRecord, currRecord models.ScanRecord, diff models.DiffReport) error

// Collect executes Run for paths and returns all discovered File entries as a slice.
func Collect(paths []string, opts models.LookOptions, errOut io.Writer) []models.File {
	ch := make(chan models.File, 256)
	go Run(paths, opts, ch, errOut)

	var files []models.File
	for f := range ch {
		files = append(files, f)
	}
	return files
}

// Watch monitors paths for filesystem changes using OS events (fsnotify), debouncing rapid
// event bursts by debounceInterval, and invokes onChange whenever entries are added, deleted,
// or modified relative to the previous snapshot. Watch blocks until ctx is canceled.
func Watch(
	ctx context.Context,
	paths []string,
	opts models.LookOptions,
	initialFiles []models.File,
	debounceInterval time.Duration,
	errOut io.Writer,
	onChange WatchCallback,
) error {
	errOut = newSyncWriter(errOut)
	if debounceInterval <= 0 {
		debounceInterval = DefaultDebounceInterval
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// Ensure hashing is enabled so snapshot diffs accurately detect content and folder Merkle changes.
	opts.Hash = true
	if opts.HashAlgo == "" {
		opts.HashAlgo = models.HashCRC32
	}

	rootPath := paths[0]
	if initialFiles == nil {
		initialFiles = Collect(paths, opts, errOut)
	}
	prevRecord := models.BuildScanRecord("baseline", "", "", rootPath, opts.HashAlgo, initialFiles)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating filesystem watcher: %w", err)
	}
	defer watcher.Close()

	absRoots := resolveWatchRoots(paths, errOut)
	registerAllWatches(watcher, absRoots, opts.ShowHidden, errOut)

	var (
		debounceTimer *time.Timer
		debounceChan  <-chan time.Time
	)
	defer func() {
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if shouldIgnoreEvent(event.Name, absRoots, opts.ShowHidden) {
				continue
			}

			// Dynamically watch newly created directories so nested changes are also captured.
			if event.Has(fsnotify.Create) {
				if info, statErr := os.Stat(event.Name); statErr == nil && info.IsDir() {
					addDirTreeWatch(watcher, event.Name, opts.ShowHidden, errOut)
				}
			}

			if debounceTimer == nil {
				debounceTimer = time.NewTimer(debounceInterval)
			} else {
				if !debounceTimer.Stop() {
					select {
					case <-debounceTimer.C:
					default:
					}
				}
				debounceTimer.Reset(debounceInterval)
			}
			debounceChan = debounceTimer.C

		case watchErr, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(errOut, "go-ls: watcher error: %v\n", watchErr)

		case <-debounceChan:
			debounceChan = nil

			// Ensure any directories created during the burst are registered with the watcher.
			registerAllWatches(watcher, absRoots, opts.ShowHidden, io.Discard)

			currFiles := Collect(paths, opts, errOut)
			currRecord := models.BuildScanRecord("live", "", "", rootPath, opts.HashAlgo, currFiles)
			diff := models.ComputeDiff(prevRecord, currRecord)
			if diff.Identical {
				continue
			}

			oldRecord := prevRecord
			prevRecord = currRecord
			if onChange != nil {
				if err := onChange(currFiles, oldRecord, currRecord, *diff); err != nil {
					return err
				}
			}
		}
	}
}

func resolveWatchRoots(paths []string, errOut io.Writer) []string {
	roots := make([]string, 0, len(paths))
	for _, rawPath := range paths {
		absPath, err := filepath.Abs(rawPath)
		if err != nil {
			fmt.Fprintf(errOut, "go-ls: cannot resolve path %q: %v\n", rawPath, err)
			continue
		}
		roots = append(roots, absPath)
	}
	return roots
}

func registerAllWatches(watcher *fsnotify.Watcher, absRoots []string, showHidden bool, errOut io.Writer) {
	for _, absPath := range absRoots {
		info, err := os.Stat(absPath)
		if err != nil {
			// If a target file does not currently exist, watch its parent directory.
			parent := filepath.Dir(absPath)
			_ = watcher.Add(parent)
			continue
		}
		if !info.IsDir() {
			// Watch parent directory so atomic file replacements (rename over target) are caught.
			if err := watcher.Add(filepath.Dir(absPath)); err != nil {
				fmt.Fprintf(errOut, "go-ls: cannot watch %q: %v\n", filepath.Dir(absPath), err)
			}
			continue
		}
		addDirTreeWatch(watcher, absPath, showHidden, errOut)
	}
}

func addDirTreeWatch(watcher *fsnotify.Watcher, rootDir string, showHidden bool, errOut io.Writer) {
	_ = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if path != rootDir && !showHidden && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if watchErr := watcher.Add(path); watchErr != nil {
			fmt.Fprintf(errOut, "go-ls: cannot watch directory %q: %v\n", path, watchErr)
		}
		return nil
	})
}

func shouldIgnoreEvent(eventPath string, absRoots []string, showHidden bool) bool {
	if showHidden {
		return false
	}
	absEvent, err := filepath.Abs(eventPath)
	if err != nil {
		absEvent = filepath.Clean(eventPath)
	}

	for _, root := range absRoots {
		rel, err := filepath.Rel(root, absEvent)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		if rel == "." {
			return false
		}
		for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
			if strings.HasPrefix(part, ".") && part != "." && part != ".." {
				return true
			}
		}
		return false
	}

	return strings.HasPrefix(filepath.Base(absEvent), ".")
}
