package cli

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/vitorecomp/go-ls/pkg/client"
	"github.com/vitorecomp/go-ls/pkg/looker"
	"github.com/vitorecomp/go-ls/pkg/models"
	"github.com/vitorecomp/go-ls/pkg/output"
	"github.com/vitorecomp/go-ls/pkg/server"
	"github.com/vitorecomp/go-ls/pkg/store"
)

const defaultChannelBuffer = 8192

// Run executes the scan command locally and optionally uploads the scan to the configured server.
func (s *ScanCmd) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return s.RunWithContext(ctx, os.Stdout, os.Stderr)
}

// RunWithContext executes the scan command using the provided context and output writers,
// supporting incremental batched server uploads, scan resumption, and continuous --live monitoring.
func (s *ScanCmd) RunWithContext(ctx context.Context, out, errOut io.Writer) error {
	lookOpts := s.ToLookOptions()
	outputOpts := s.ToOutputOptions()

	isResume := strings.TrimSpace(s.Resume) != ""
	scanID := strings.TrimSpace(s.ScanID)
	if isResume {
		scanID = strings.TrimSpace(s.Resume)
	}
	if scanID == "" {
		scanID = models.GenerateScanID()
	}

	serverURL := strings.TrimSpace(s.ServerURL)
	paths := append([]string(nil), s.Paths...)
	if len(paths) == 0 {
		paths = []string{"."}
	}
	tag := s.Tag

	knownServerEntries := make(map[string]models.ScanEntry)
	lookOpts.KnownFiles = make(map[string]models.KnownFileMetadata)

	var localCP *localScanCheckpoint
	if isResume {
		if cp, err := loadLocalCheckpoint(scanID); err == nil && cp != nil {
			localCP = cp
			if serverURL == "" && cp.ServerURL != "" {
				serverURL = cp.ServerURL
			}
			if len(paths) == 1 && paths[0] == "." && cp.RootPath != "" {
				paths = []string{cp.RootPath}
			}
			if !s.Recursive && cp.Recursive {
				lookOpts.Recursive = true
			}
			if !s.ShowHidden && cp.ShowHidden {
				lookOpts.ShowHidden = true
				outputOpts.ShowHidden = true
			}
			if cp.HashAlgo != "" {
				lookOpts.HashAlgo = models.HashAlgorithm(cp.HashAlgo)
			}
			if tag == "" && cp.Tag != "" {
				tag = cp.Tag
			}
			for _, e := range cp.Entries {
				if e.AbsPath != "" && e.Hash != "" && !e.IsDir {
					lookOpts.KnownFiles[e.AbsPath] = models.KnownFileMetadata{
						Size:    e.Size,
						ModTime: e.ModTime,
						Hash:    e.Hash,
					}
				}
			}
		} else if serverURL == "" {
			serverURL = DefaultServerURL
		}
	}

	var apiClient *client.Client
	if serverURL != "" {
		apiClient = client.New(serverURL)
		lookOpts.Hash = true
		outputOpts.Hash = true
	}

	if isResume && apiClient != nil {
		remoteScan, err := apiClient.GetScan(ctx, scanID)
		if err != nil {
			if localCP == nil {
				return fmt.Errorf("resuming scan %s from %s: %w", scanID, serverURL, err)
			}
		} else if remoteScan != nil {
			scanID = remoteScan.ID
			if len(paths) == 1 && paths[0] == "." && remoteScan.RootPath != "" {
				paths = []string{remoteScan.RootPath}
			}
			if remoteScan.HashAlgo != "" {
				lookOpts.HashAlgo = models.HashAlgorithm(remoteScan.HashAlgo)
			}
			if tag == "" && remoteScan.Tag != "" {
				tag = remoteScan.Tag
			}
			for _, e := range remoteScan.Entries {
				knownServerEntries[e.RelPath] = e
				if e.IsDir || strings.Contains(e.RelPath, "/") {
					lookOpts.Recursive = true
				}
				if !e.IsDir && e.Hash != "" {
					absPath := e.AbsPath
					if absPath == "" && remoteScan.RootPath != "" {
						absPath = filepath.Join(remoteScan.RootPath, filepath.FromSlash(e.RelPath))
					}
					if absPath != "" {
						lookOpts.KnownFiles[absPath] = models.KnownFileMetadata{
							Size:    e.Size,
							ModTime: e.ModTime,
							Hash:    e.Hash,
						}
					}
				}
			}
		}
	}

	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown-host"
	}
	computerID := models.DetectComputerID(hostname)

	rootPath := "."
	if len(paths) > 0 {
		rootPath = paths[0]
	}
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		absRoot = filepath.Clean(rootPath)
	}
	driveID := models.DetectDriveID(absRoot)

	batchSize := s.BatchSize
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}

	// Initialize the scan session on the server before walking the directory so the
	// canonical scan ID is resolved and ready for resumption immediately.
	if apiClient != nil && !isResume {
		initSummary, err := apiClient.UploadScanBatch(ctx, models.ScanBatchRequest{
			ID:         scanID,
			Tag:        tag,
			ComputerID: computerID,
			Hostname:   hostname,
			DriveID:    driveID,
			RootPath:   absRoot,
			HashAlgo:   string(lookOpts.HashAlgo),
			BatchIndex: 0,
			Final:      false,
			Resume:     false,
		})
		if err != nil {
			return fmt.Errorf("initializing scan on %s: %w", serverURL, err)
		}
		if initSummary != nil && initSummary.ID != "" {
			scanID = initSummary.ID
		}
	}

	// Print the scan ID as the very first output line so it can be used to resume the scan.
	if s.JSON || s.Compare {
		fmt.Fprintf(errOut, "Scan ID: %s\n", scanID)
	} else {
		fmt.Fprintf(out, "Scan ID: %s\n", scanID)
	}

	if localCP == nil {
		localCP = &localScanCheckpoint{
			ID:         scanID,
			ServerURL:  serverURL,
			RootPath:   absRoot,
			Recursive:  lookOpts.Recursive,
			ShowHidden: lookOpts.ShowHidden,
			HashAlgo:   string(lookOpts.HashAlgo),
			Tag:        tag,
			Entries:    make(map[string]models.ScanEntry),
		}
		for k, v := range knownServerEntries {
			localCP.Entries[k] = v
		}
	} else {
		localCP.ID = scanID
		localCP.ServerURL = serverURL
		localCP.RootPath = absRoot
		localCP.Completed = false
	}
	saveLocalCheckpoint(localCP)

	var progressBar *output.ProgressBar
	lookErrOut := errOut
	displayOut := out
	if lookOpts.Recursive && !s.NoProgress {
		progressBar = output.NewProgressBar(errOut, outputOpts.NoColor)
		lookOpts.OnProgress = progressBar.Update
		lookErrOut = progressBar.WrapStderr(errOut)
		displayOut = progressBar.WrapStdout(out)
	}

	rawChan := make(models.OutputChan, defaultChannelBuffer)
	go looker.RunWithContext(ctx, paths, lookOpts, rawChan, lookErrOut)

	var (
		collected    []models.File
		pendingBatch = make([]models.ScanEntry, 0, batchSize)
		batchIndex   int
		displayChan  chan models.File
		displayDone  chan struct{}
	)
	if !s.Quiet || apiClient == nil {
		displayChan = make(chan models.File, defaultChannelBuffer)
		displayDone = make(chan struct{})
		go func() {
			defer close(displayDone)
			output.WriteOutput(displayOut, displayChan, outputOpts)
		}()
	}

	flushBatch := func(reqCtx context.Context, entries []models.ScanEntry, final bool, rootHash string) (*models.ScanSummary, error) {
		for _, e := range entries {
			localCP.Entries[e.RelPath] = e
		}
		if final {
			localCP.Completed = true
		}
		saveLocalCheckpoint(localCP)

		if apiClient == nil {
			return nil, nil
		}
		batchIndex++
		return apiClient.UploadScanBatch(reqCtx, models.ScanBatchRequest{
			ID:         scanID,
			Tag:        tag,
			ComputerID: computerID,
			Hostname:   hostname,
			DriveID:    driveID,
			RootPath:   absRoot,
			HashAlgo:   string(lookOpts.HashAlgo),
			RootHash:   rootHash,
			Entries:    entries,
			BatchIndex: batchIndex,
			Final:      final,
			Resume:     isResume,
		})
	}

	var batchErr error
	for f := range rawChan {
		collected = append(collected, f)
		if displayChan != nil {
			displayChan <- f
		}

		entry := models.FileToScanEntry(absRoot, f)
		localCP.Entries[entry.RelPath] = entry

		if isResume {
			if prev, ok := knownServerEntries[entry.RelPath]; ok &&
				prev.Hash == entry.Hash &&
				prev.Size == entry.Size &&
				prev.IsDir == entry.IsDir &&
				prev.Hash != "" {
				continue
			}
		}

		pendingBatch = append(pendingBatch, entry)
		if len(pendingBatch) >= batchSize && batchErr == nil && ctx.Err() == nil {
			if _, err := flushBatch(ctx, pendingBatch, false, ""); err != nil {
				batchErr = fmt.Errorf("uploading scan batch to %s: %w", serverURL, err)
			}
			pendingBatch = pendingBatch[:0]
		}
	}

	if displayChan != nil {
		close(displayChan)
		<-displayDone
	}

	if progressBar != nil {
		progressBar.Finish()
	}

	if batchErr != nil {
		return batchErr
	}

	if ctx.Err() != nil {
		if len(pendingBatch) > 0 {
			flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, _ = flushBatch(flushCtx, pendingBatch, false, "")
			cancel()
		} else {
			saveLocalCheckpoint(localCP)
		}
		fmt.Fprintf(errOut, "Scan interrupted. Resume with: go-ls scan --resume=%s\n", scanID)
		return ctx.Err()
	}

	if apiClient != nil {
		record := models.BuildScanRecord(scanID, tag, hostname, absRoot, lookOpts.HashAlgo, collected)
		summary, err := flushBatch(ctx, pendingBatch, true, record.RootHash)
		if err != nil {
			return fmt.Errorf("finalizing scan on %s: %w", serverURL, err)
		}
		output.WriteScanSummary(out, *summary, s.JSON)
	} else {
		_, _ = flushBatch(ctx, pendingBatch, true, "")
	}

	if !s.Live {
		return nil
	}

	fmt.Fprintf(errOut, "Watching %v for changes (press Ctrl+C to stop)...\n", paths)

	watchOpts := lookOpts
	watchOpts.OnProgress = nil
	watchOpts.KnownFiles = nil

	return looker.Watch(
		ctx,
		paths,
		watchOpts,
		collected,
		looker.DefaultDebounceInterval,
		errOut,
		func(files []models.File, _, currRecord models.ScanRecord, diff models.DiffReport) error {
			var (
				uploadedScanID  string
				uploadedVersion int
			)

			if apiClient != nil {
				record := models.BuildScanRecord(scanID, tag, hostname, absRoot, lookOpts.HashAlgo, files)
				summary, err := apiClient.UploadScan(ctx, record)
				if err != nil {
					return fmt.Errorf("uploading live scan to %s: %w", serverURL, err)
				}
				uploadedScanID = summary.ID
				uploadedVersion = summary.Version
			}

			event := models.LiveEvent{
				Timestamp:   time.Now().UTC(),
				RootPath:    currRecord.RootPath,
				RootHash:    currRecord.RootHash,
				Added:       diff.Added,
				Deleted:     diff.Deleted,
				Modified:    diff.Modified,
				Changes:     diff.Changes,
				ScanID:      uploadedScanID,
				ScanVersion: uploadedVersion,
			}
			output.WriteLiveEvent(out, event, s.JSON, outputOpts.NoColor)
			return nil
		},
	)
}

// Run executes the scans list/inspect command against the remote server.
func (s *ScansCmd) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	apiClient := client.New(s.ServerURL)

	if s.ScanID == "" {
		summaries, err := apiClient.ListScans(ctx)
		if err != nil {
			return err
		}
		output.WriteScansList(os.Stdout, summaries, s.JSON)
		return nil
	}

	rec, err := apiClient.GetScan(ctx, s.ScanID)
	if err != nil {
		return err
	}
	output.WriteScanDetails(os.Stdout, *rec, s.JSON)
	return nil
}

// Run executes the diff command between two scans stored on the server.
func (d *DiffCmd) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	apiClient := client.New(d.ServerURL)
	report, err := apiClient.DiffScans(ctx, d.BaseID, d.TargetID)
	if err != nil {
		return err
	}
	output.WriteDiffReport(os.Stdout, *report, d.JSON, d.Color == "never")
	return nil
}

// Run executes the duplicates command against the remote server.
func (d *DuplicatesCmd) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	apiClient := client.New(d.ServerURL)
	groups, err := apiClient.FindDuplicates(ctx, d.ScanID)
	if err != nil {
		return err
	}
	output.WriteDuplicates(os.Stdout, groups, d.JSON)
	return nil
}

// Run starts the go-ls HTTP server and blocks until interrupted by SIGINT or SIGTERM.
func (s *ServeCmd) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if s.Dev && os.Getenv("GOLS_DEV_CHILD") == "" {
		return s.runDevSupervisor(ctx)
	}

	st, err := store.Open(ctx, s.DBURL)
	if err != nil {
		return fmt.Errorf("initializing storage backend: %w", err)
	}
	defer st.Close()

	httpServer := &http.Server{
		Addr:              s.ListenAddr,
		Handler:           server.NewWithUIAndProxy(st, s.UIDir, s.UIProxyURL),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errChan := make(chan error, 1)
	go func() {
		log.Printf("go-ls server v%s listening on %s (storage: %s)", Version, s.ListenAddr, redactDBURL(s.DBURL))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
		close(errChan)
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		log.Printf("shutting down go-ls server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	}
}

// Run prints the version string to stdout.
func (v *VersionCmd) Run() error {
	fmt.Printf("go-ls v%s\n", Version)
	return nil
}

func redactDBURL(dbURL string) string {
	if dbURL == "" || dbURL == "memory://" || dbURL == "memory" {
		return "memory://"
	}
	return "postgres"
}
