package cli

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
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
// supporting both one-shot scans and continuous --live monitoring.
func (s *ScanCmd) RunWithContext(ctx context.Context, out, errOut io.Writer) error {
	lookOpts := s.ToLookOptions()
	outputOpts := s.ToOutputOptions()

	var progressBar *output.ProgressBar
	lookErrOut := errOut
	displayOut := out
	if s.Recursive && !s.NoProgress {
		progressBar = output.NewProgressBar(errOut, outputOpts.NoColor)
		lookOpts.OnProgress = progressBar.Update
		lookErrOut = progressBar.WrapStderr(errOut)
		displayOut = progressBar.WrapStdout(out)
	}

	rawChan := make(models.OutputChan, defaultChannelBuffer)
	go looker.Run(s.Paths, lookOpts, rawChan, lookErrOut)

	// Pure one-shot local mode when neither --server nor --live is specified.
	if s.ServerURL == "" && !s.Live {
		output.WriteOutput(displayOut, rawChan, outputOpts)
		if progressBar != nil {
			progressBar.Finish()
		}
		return nil
	}

	// Collect discovered files for server upload and/or live monitoring baseline,
	// teeing to local output unless suppressed by --quiet with --server.
	var (
		collected   []models.File
		displayChan chan models.File
		displayDone chan struct{}
	)
	if !s.Quiet || s.ServerURL == "" {
		displayChan = make(chan models.File, defaultChannelBuffer)
		displayDone = make(chan struct{})
		go func() {
			defer close(displayDone)
			output.WriteOutput(displayOut, displayChan, outputOpts)
		}()
	}

	for f := range rawChan {
		collected = append(collected, f)
		if displayChan != nil {
			displayChan <- f
		}
	}

	if displayChan != nil {
		close(displayChan)
		<-displayDone
	}

	if progressBar != nil {
		progressBar.Finish()
	}

	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown-host"
	}

	rootPath := "."
	if len(s.Paths) > 0 {
		rootPath = s.Paths[0]
	}

	var apiClient *client.Client
	if s.ServerURL != "" {
		apiClient = client.New(s.ServerURL)
		record := models.BuildScanRecord(s.ScanID, s.Tag, hostname, rootPath, lookOpts.HashAlgo, collected)
		summary, err := apiClient.UploadScan(ctx, record)
		if err != nil {
			return fmt.Errorf("uploading scan to %s: %w", s.ServerURL, err)
		}
		output.WriteScanSummary(out, *summary, s.JSON)
	}

	if !s.Live {
		return nil
	}

	fmt.Fprintf(errOut, "Watching %v for changes (press Ctrl+C to stop)...\n", s.Paths)

	watchOpts := lookOpts
	watchOpts.OnProgress = nil

	return looker.Watch(
		ctx,
		s.Paths,
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
				record := models.BuildScanRecord(s.ScanID, s.Tag, hostname, rootPath, lookOpts.HashAlgo, files)
				summary, err := apiClient.UploadScan(ctx, record)
				if err != nil {
					return fmt.Errorf("uploading live scan to %s: %w", s.ServerURL, err)
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
