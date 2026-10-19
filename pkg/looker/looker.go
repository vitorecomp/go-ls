// Package looker provides concurrent directory traversal and file/directory hashing.
package looker

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/vitorecomp/go-ls/pkg/models"
)

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
	errOut = newSyncWriter(errOut)
	sem := newSemaphore(opts.Workers)

	var wg sync.WaitGroup
	for _, rawPath := range paths {
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
			f := models.NewFile(filepath.Dir(absPath), info)
			if opts.Hash {
				withSemaphore(sem, func() {
					if err := f.ComputeHash(opts.HashAlgo); err != nil {
						fmt.Fprintf(errOut, "go-ls: %v\n", err)
					}
				})
			}
			out <- f
			continue
		}

		wg.Add(1)
		go func(dirPath string) {
			defer wg.Done()
			walkDir(sem, dirPath, opts, true, out, errOut)
		}(absPath)
	}

	wg.Wait()
	close(out)
}

// Look traverses a single directory path and sends discovered entries to channel,
// decrementing wg when finished.
func Look(wg *sync.WaitGroup, path string, parameters models.LookParameters, channel chan<- models.File) {
	defer wg.Done()
	sem := newSemaphore(parameters.Workers)
	walkDir(sem, path, parameters, true, channel, newSyncWriter(os.Stderr))
}

// walkDir scans dirPath, optionally emits discovered entries to out, and returns the
// combined directory hash when opts.Hash is enabled.
func walkDir(
	sem chan struct{},
	dirPath string,
	opts models.LookOptions,
	emit bool,
	out chan<- models.File,
	errOut io.Writer,
) string {
	var entries []os.DirEntry
	var readErr error
	withSemaphore(sem, func() {
		entries, readErr = os.ReadDir(dirPath)
	})
	if readErr != nil {
		fmt.Fprintf(errOut, "go-ls: cannot read directory %q: %v\n", dirPath, readErr)
		return ""
	}

	items := make([]models.File, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !opts.ShowHidden && strings.HasPrefix(name, ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			fmt.Fprintf(errOut, "go-ls: cannot stat %q in %q: %v\n", name, dirPath, err)
			continue
		}

		items = append(items, models.NewFile(dirPath, info))
	}

	// Fast path when hashing is disabled: stream entries immediately.
	if !opts.Hash {
		var subWg sync.WaitGroup
		for _, item := range items {
			if item.Dir && opts.Recursive {
				subPath := item.GetAbsolutePath()
				subWg.Add(1)
				go func(p string) {
					defer subWg.Done()
					walkDir(sem, p, opts, true, out, errOut)
				}(subPath)
			}
			if emit {
				out <- item
			}
		}
		subWg.Wait()
		return ""
	}

	// Hashing path: compute file hashes and subdirectory Merkle hashes concurrently,
	// preserving os.ReadDir's deterministic sorted order when combining child hashes.
	var childWg sync.WaitGroup
	for i := range items {
		childWg.Add(1)
		go func(idx int) {
			defer childWg.Done()
			if items[idx].Dir {
				subPath := items[idx].GetAbsolutePath()
				items[idx].Hash = walkDir(sem, subPath, opts, emit && opts.Recursive, out, errOut)
				return
			}
			withSemaphore(sem, func() {
				if err := items[idx].ComputeHash(opts.HashAlgo); err != nil {
					fmt.Fprintf(errOut, "go-ls: %v\n", err)
				}
			})
		}(i)
	}
	childWg.Wait()

	childHashes := make([]string, 0, len(items))
	for _, item := range items {
		childHashes = append(childHashes, item.Hash)
		if emit {
			out <- item
		}
	}

	var dirHolder models.File
	return dirHolder.ComputeDirHash(opts.HashAlgo, childHashes)
}

func newSemaphore(workers int) chan struct{} {
	if workers <= 0 {
		workers = runtime.NumCPU()
		if workers < 4 {
			workers = 4
		}
	}
	return make(chan struct{}, workers)
}

func withSemaphore(sem chan struct{}, fn func()) {
	sem <- struct{}{}
	defer func() { <-sem }()
	fn()
}
