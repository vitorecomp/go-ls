// Package looker provides concurrent directory traversal and file/directory hashing.
package looker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitorecomp/go-ls/pkg/models"
)

const progressEmitIntervalNs = int64(25 * time.Millisecond)

type progressTracker struct {
	fn         models.ProgressFunc
	discovered atomic.Int64
	completed  atomic.Int64
	dirs       atomic.Int64
	bytes      atomic.Int64
	lastEmitNs atomic.Int64
	lastPath   atomic.Value // string
	mu         sync.Mutex
}

func newProgressTracker(fn models.ProgressFunc) *progressTracker {
	if fn == nil {
		return nil
	}
	return &progressTracker{fn: fn}
}

func (p *progressTracker) addDir(path string) {
	if p == nil {
		return
	}
	p.dirs.Add(1)
	p.maybeEmit(path, false)
}

func (p *progressTracker) addDiscovered(n int64, path string) {
	if p == nil || n <= 0 {
		return
	}
	p.discovered.Add(n)
	p.maybeEmit(path, false)
}

func (p *progressTracker) addCompleted(size int64, path string) {
	if p == nil {
		return
	}
	if size > 0 {
		p.bytes.Add(size)
	}
	p.completed.Add(1)
	p.maybeEmit(path, false)
}

func (p *progressTracker) finish() {
	if p == nil {
		return
	}
	p.maybeEmit("", true)
}

func (p *progressTracker) maybeEmit(current string, done bool) {
	if current != "" {
		p.lastPath.Store(current)
	}
	if !done {
		now := time.Now().UnixNano()
		prev := p.lastEmitNs.Load()
		if prev != 0 && now-prev < progressEmitIntervalNs {
			return
		}
		if !p.lastEmitNs.CompareAndSwap(prev, now) {
			return
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	disc := p.discovered.Load()
	comp := p.completed.Load()
	if done && comp > disc {
		disc = comp
	}
	if current == "" {
		if v, ok := p.lastPath.Load().(string); ok {
			current = v
		}
	}
	p.fn(models.ProgressSnapshot{
		Discovered: disc,
		Completed:  comp,
		Dirs:       p.dirs.Load(),
		Bytes:      p.bytes.Load(),
		Current:    current,
		Done:       done,
	})
}

func itemByteSize(f models.File) int64 {
	if f.FileInfo == nil || f.Dir {
		return 0
	}
	return f.FileInfo.Size()
}

func lookupKnownFileHash(opts models.LookOptions, absPath string, f models.File) (string, bool) {
	if len(opts.KnownFiles) == 0 || f.FileInfo == nil || f.Dir {
		return "", false
	}
	known, ok := opts.KnownFiles[absPath]
	if !ok || known.Hash == "" {
		return "", false
	}
	if known.Size != f.FileInfo.Size() {
		return "", false
	}
	if !known.ModTime.IsZero() && !f.FileInfo.ModTime().UTC().Truncate(time.Millisecond).Equal(known.ModTime.UTC().Truncate(time.Millisecond)) {
		return "", false
	}
	return known.Hash, true
}

type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func newSyncWriter(w io.Writer) io.Writer {
	if w == nil || w == io.Discard {
		return io.Discard
	}
	if _, ok := w.(*syncWriter); ok {
		return w
	}
	return &syncWriter{w: w}
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}

// Run traverses all provided paths (files or directories) concurrently according to opts,
// streaming discovered models.File entries to out and closing out when traversal finishes.
func Run(paths []string, opts models.LookOptions, out chan<- models.File, errOut io.Writer) {
	RunWithContext(context.Background(), paths, opts, out, errOut)
}

// RunWithContext traverses all provided paths concurrently until traversal finishes or ctx is canceled,
// streaming discovered models.File entries to out and closing out when done.
func RunWithContext(ctx context.Context, paths []string, opts models.LookOptions, out chan<- models.File, errOut io.Writer) {
	if ctx == nil {
		ctx = context.Background()
	}
	errOut = newSyncWriter(errOut)
	dirSem := newSemaphore(opts.Workers)
	hashSem := newSemaphore(opts.Workers)
	tracker := newProgressTracker(opts.OnProgress)

	var wg sync.WaitGroup
	for _, rawPath := range paths {
		if ctx.Err() != nil {
			break
		}
		absPath, err := filepath.Abs(rawPath)
		if err != nil {
			fmt.Fprintf(errOut, "go-ls: cannot resolve path %q: %v\n", rawPath, err)
			continue
		}

		info, err := os.Stat(absPath)
		if err != nil {
			fmt.Fprintf(errOut, "go-ls: cannot access %q: %v\n", rawPath, err)
			continue
		}

		if !info.IsDir() {
			parentDir := filepath.Dir(absPath)
			tracker.addDiscovered(1, parentDir)
			f := models.NewFile(parentDir, info)
			if opts.Hash {
				if h, ok := lookupKnownFileHash(opts, absPath, f); ok {
					f.Hash = h
				} else {
					withSemaphore(hashSem, func() {
						if err := f.ComputeHash(opts.HashAlgo); err != nil {
							fmt.Fprintf(errOut, "go-ls: %v\n", err)
						}
					})
				}
			}
			tracker.addCompleted(itemByteSize(f), parentDir)
			out <- f
			continue
		}

		wg.Add(1)
		go func(dirPath string) {
			defer wg.Done()
			walkDir(ctx, dirSem, hashSem, dirPath, opts, true, out, errOut, tracker)
		}(absPath)
	}

	wg.Wait()
	tracker.finish()
	close(out)
}

// Look traverses a single directory path and sends discovered entries to channel,
// decrementing wg when finished.
func Look(wg *sync.WaitGroup, path string, parameters models.LookParameters, channel chan<- models.File) {
	defer wg.Done()
	dirSem := newSemaphore(parameters.Workers)
	hashSem := newSemaphore(parameters.Workers)
	tracker := newProgressTracker(parameters.OnProgress)
	walkDir(context.Background(), dirSem, hashSem, path, parameters, true, channel, newSyncWriter(os.Stderr), tracker)
	tracker.finish()
}

// walkDir scans dirPath, optionally emits discovered entries to out, and returns the
// combined directory hash when opts.Hash is enabled.
func walkDir(
	ctx context.Context,
	dirSem chan struct{},
	hashSem chan struct{},
	dirPath string,
	opts models.LookOptions,
	emit bool,
	out chan<- models.File,
	errOut io.Writer,
	tracker *progressTracker,
) string {
	if ctx.Err() != nil {
		return ""
	}
	tracker.addDir(dirPath)

	var entries []os.DirEntry
	var readErr error
	withSemaphore(dirSem, func() {
		entries, readErr = os.ReadDir(dirPath)
	})
	if readErr != nil {
		fmt.Fprintf(errOut, "go-ls: cannot read directory %q: %v\n", dirPath, readErr)
		return ""
	}

	visibleEntries := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if !opts.ShowHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		visibleEntries = append(visibleEntries, entry)
	}

	items := buildDirItems(dirPath, visibleEntries, errOut)
	tracker.addDiscovered(int64(len(items)), dirPath)

	// Fast path when hashing is disabled: fan out subdirectories immediately across CPU cores,
	// then stream entries in parallel.
	if !opts.Hash {
		var subWg sync.WaitGroup
		if opts.Recursive {
			for i := range items {
				if ctx.Err() != nil {
					break
				}
				if items[i].Dir {
					subPath := items[i].GetAbsolutePath()
					subWg.Add(1)
					go func(p string) {
						defer subWg.Done()
						walkDir(ctx, dirSem, hashSem, p, opts, true, out, errOut, tracker)
					}(subPath)
				}
			}
		}
		for _, item := range items {
			if ctx.Err() != nil {
				break
			}
			tracker.addCompleted(itemByteSize(item), dirPath)
			if emit {
				out <- item
			}
		}
		subWg.Wait()
		return ""
	}

	// Hashing path: compute file hashes and subdirectory Merkle hashes concurrently,
	// streaming each item to out as soon as its hash is ready while preserving
	// os.ReadDir's deterministic sorted order when combining child hashes.
	var childWg sync.WaitGroup
	for i := range items {
		childWg.Add(1)
		go func(idx int) {
			defer childWg.Done()
			if ctx.Err() != nil {
				return
			}
			absItemPath := items[idx].GetAbsolutePath()
			if items[idx].Dir {
				items[idx].Hash = walkDir(ctx, dirSem, hashSem, absItemPath, opts, emit && opts.Recursive, out, errOut, tracker)
				if ctx.Err() != nil {
					return
				}
				tracker.addCompleted(0, dirPath)
				if emit {
					out <- items[idx]
				}
				return
			}
			if h, ok := lookupKnownFileHash(opts, absItemPath, items[idx]); ok {
				items[idx].Hash = h
			} else {
				withSemaphore(hashSem, func() {
					if ctx.Err() != nil {
						return
					}
					if err := items[idx].ComputeHash(opts.HashAlgo); err != nil {
						fmt.Fprintf(errOut, "go-ls: %v\n", err)
					}
				})
			}
			if ctx.Err() != nil {
				return
			}
			tracker.addCompleted(itemByteSize(items[idx]), dirPath)
			if emit {
				out <- items[idx]
			}
		}(i)
	}
	childWg.Wait()

	if ctx.Err() != nil {
		return ""
	}
	childHashes := make([]string, 0, len(items))
	for _, item := range items {
		childHashes = append(childHashes, item.Hash)
	}

	var dirHolder models.File
	return dirHolder.ComputeDirHash(opts.HashAlgo, childHashes)
}

func buildDirItems(dirPath string, visibleEntries []os.DirEntry, errOut io.Writer) []models.File {
	n := len(visibleEntries)
	if n == 0 {
		return nil
	}

	numCPU := runtime.GOMAXPROCS(0)
	if n < 64 || numCPU <= 1 {
		items := make([]models.File, 0, n)
		for _, entry := range visibleEntries {
			info, err := entry.Info()
			if err != nil {
				fmt.Fprintf(errOut, "go-ls: cannot stat %q in %q: %v\n", entry.Name(), dirPath, err)
				continue
			}
			items = append(items, models.NewFile(dirPath, info))
		}
		return items
	}

	slots := make([]models.File, n)
	valid := make([]bool, n)
	var wg sync.WaitGroup
	chunkSize := (n + numCPU - 1) / numCPU
	for w := 0; w < numCPU; w++ {
		start := w * chunkSize
		if start >= n {
			break
		}
		end := start + chunkSize
		if end > n {
			end = n
		}
		wg.Add(1)
		go func(lo, hi int) {
			defer wg.Done()
			for idx := lo; idx < hi; idx++ {
				entry := visibleEntries[idx]
				info, err := entry.Info()
				if err != nil {
					fmt.Fprintf(errOut, "go-ls: cannot stat %q in %q: %v\n", entry.Name(), dirPath, err)
					continue
				}
				slots[idx] = models.NewFile(dirPath, info)
				valid[idx] = true
			}
		}(start, end)
	}
	wg.Wait()

	items := make([]models.File, 0, n)
	for idx := 0; idx < n; idx++ {
		if valid[idx] {
			items = append(items, slots[idx])
		}
	}
	return items
}

// DefaultWorkers returns the optimal parallel worker count for the current machine,
// scaling with all available logical CPUs to saturate both CPU hashing and filesystem I/O.
func DefaultWorkers() int {
	cpus := runtime.NumCPU()
	if runtime.GOMAXPROCS(0) < cpus {
		runtime.GOMAXPROCS(cpus)
	}
	workers := cpus * 4
	if workers < 16 {
		workers = 16
	}
	return workers
}

func newSemaphore(workers int) chan struct{} {
	if workers <= 0 {
		workers = DefaultWorkers()
	}
	return make(chan struct{}, workers)
}

func withSemaphore(sem chan struct{}, fn func()) {
	sem <- struct{}{}
	defer func() { <-sem }()
	fn()
}
