package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/vitorecomp/go-ls/pkg/models"
)

// WriteScanSummary renders a newly uploaded or inspected ScanSummary.
func WriteScanSummary(w io.Writer, summary models.ScanSummary, asJSON bool) {
	if asJSON {
		enc := json.NewEncoder(w)
		_ = enc.Encode(summary)
		return
	}
	fmt.Fprintf(w, "Scan stored on server:\n")
	fmt.Fprintf(w, "  ID:         %s (v%d)\n", summary.ID, max(summary.Version, 1))
	if summary.Tag != "" {
		fmt.Fprintf(w, "  Tag:        %s\n", summary.Tag)
	}
	compLabel := summary.Hostname
	if summary.ComputerID != "" && summary.ComputerID != summary.Hostname {
		compLabel = fmt.Sprintf("%s (%s)", summary.Hostname, summary.ComputerID)
	}
	fmt.Fprintf(w, "  Computer:   %s\n", compLabel)
	if summary.DriveID != "" {
		fmt.Fprintf(w, "  Drive ID:   %s\n", summary.DriveID)
	}
	fmt.Fprintf(w, "  Root Path:  %s\n", summary.RootPath)
	fmt.Fprintf(w, "  Root Hash:  %s (%s)\n", summary.RootHash, summary.HashAlgo)
	fmt.Fprintf(w, "  Files/Dirs: %d files, %d dirs (%d bytes)\n",
		summary.TotalFiles, summary.TotalDirs, summary.TotalBytes)
}

// WriteScansList renders a table of stored scan summaries.
func WriteScansList(w io.Writer, summaries []models.ScanSummary, asJSON bool) {
	if asJSON {
		enc := json.NewEncoder(w)
		_ = enc.Encode(summaries)
		return
	}
	if len(summaries) == 0 {
		fmt.Fprintln(w, "No scans stored on server.")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tVER\tTAG\tCOMPUTER\tDRIVE ID\tROOT PATH\tROOT HASH\tFILES\tDIRS\tBYTES\tUPDATED AT")
	for _, s := range summaries {
		tag := s.Tag
		if tag == "" {
			tag = "-"
		}
		drive := s.DriveID
		if drive == "" {
			drive = "-"
		}
		ts := s.UpdatedAt
		if ts.IsZero() {
			ts = s.CreatedAt
		}
		fmt.Fprintf(tw, "%s\tv%d\t%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%s\n",
			s.ID,
			max(s.Version, 1),
			tag,
			s.Hostname,
			drive,
			s.RootPath,
			s.RootHash,
			s.TotalFiles,
			s.TotalDirs,
			s.TotalBytes,
			ts.Format(time.RFC3339),
		)
	}
	_ = tw.Flush()
}

// WriteScanDetails renders a stored ScanRecord including its change history and entries.
func WriteScanDetails(w io.Writer, rec models.ScanRecord, asJSON bool) {
	if asJSON {
		enc := json.NewEncoder(w)
		_ = enc.Encode(rec)
		return
	}

	WriteScanSummary(w, rec.ScanSummary, false)

	if len(rec.History) > 0 {
		fmt.Fprintf(w, "\nDirectory Change History (%d revision(s)):\n", len(rec.History))
		htw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(htw, "VERSION\tROOT HASH\tCHANGES\tFILES\tBYTES\tTIMESTAMP")
		for _, h := range rec.History {
			hashTransition := h.RootHash
			if h.PreviousHash != "" {
				hashTransition = fmt.Sprintf("%s -> %s", h.PreviousHash, h.RootHash)
			}
			fmt.Fprintf(htw, "v%d\t%s\t+%d / -%d / ~%d\t%d\t%d\t%s\n",
				h.Version,
				hashTransition,
				h.Added,
				h.Deleted,
				h.Modified,
				h.TotalFiles,
				h.TotalBytes,
				h.CreatedAt.Format(time.RFC3339),
			)
		}
		_ = htw.Flush()
	}

	fmt.Fprintln(w)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "MODE\tOWNER\tGROUP\tSIZE\tHASH\tREL PATH")
	for _, e := range rec.Entries {
		h := e.Hash
		if h == "" {
			h = "-"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\t%s\n",
			e.Mode, e.Owner, e.Group, e.Size, h, e.RelPath)
	}
	_ = tw.Flush()
}

// WriteDiffReport renders a comparison report between two stored scans.
func WriteDiffReport(w io.Writer, report models.DiffReport, asJSON bool, noColor bool) {
	if asJSON {
		enc := json.NewEncoder(w)
		_ = enc.Encode(report)
		return
	}

	fmt.Fprintf(w, "Diff Report: %s (%s:%s) -> %s (%s:%s)\n",
		report.BaseScan.ID, report.BaseScan.Hostname, report.BaseScan.RootPath,
		report.TargetScan.ID, report.TargetScan.Hostname, report.TargetScan.RootPath,
	)
	fmt.Fprintf(w, "Summary: +%d added, -%d deleted, ~%d modified, =%d unchanged\n",
		report.Added, report.Deleted, report.Modified, report.Unchanged,
	)

	if report.Identical {
		fmt.Fprintln(w, "Scans are identical.")
		return
	}

	fmt.Fprintln(w)
	writeDiffItems(w, report.Changes, noColor)
}

// WriteLiveEvent renders an incremental batch of filesystem changes detected in --live mode.
func WriteLiveEvent(w io.Writer, event models.LiveEvent, asJSON bool, noColor bool) {
	if asJSON {
		enc := json.NewEncoder(w)
		_ = enc.Encode(event)
		return
	}

	ts := event.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	fmt.Fprintf(w, "\n[%s] Change detected (+%d added, -%d deleted, ~%d modified) | Root Hash: %s\n",
		ts.Local().Format("15:04:05"),
		event.Added,
		event.Deleted,
		event.Modified,
		event.RootHash,
	)
	writeDiffItems(w, event.Changes, noColor)
	if event.ScanID != "" {
		if event.ScanVersion > 0 {
			fmt.Fprintf(w, "Scan stored on server: %s (v%d)\n", event.ScanID, event.ScanVersion)
		} else {
			fmt.Fprintf(w, "Scan stored on server: %s\n", event.ScanID)
		}
	}
}

func writeDiffItems(w io.Writer, changes []models.DiffItem, noColor bool) {
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)
	if noColor {
		green.DisableColor()
		red.DisableColor()
		yellow.DisableColor()
	}

	for _, c := range changes {
		kind := "file"
		if c.IsDir {
			kind = "dir"
		}
		switch c.Change {
		case models.DiffAdded:
			green.Fprintf(w, "+ [ADDED]    (%s) %s (hash=%s size=%d)\n",
				kind, c.RelPath, c.TargetHash, c.TargetSize)
		case models.DiffDeleted:
			red.Fprintf(w, "- [DELETED]  (%s) %s (hash=%s size=%d)\n",
				kind, c.RelPath, c.BaseHash, c.BaseSize)
		case models.DiffModified:
			yellow.Fprintf(w, "~ [MODIFIED] (%s) %s (hash: %s -> %s, size: %d -> %d)\n",
				kind, c.RelPath, c.BaseHash, c.TargetHash, c.BaseSize, c.TargetSize)
		}
	}
}

// WriteDuplicates renders duplicate file groups discovered on the server.
func WriteDuplicates(w io.Writer, groups []models.DuplicateGroup, asJSON bool) {
	if asJSON {
		enc := json.NewEncoder(w)
		_ = enc.Encode(groups)
		return
	}

	if len(groups) == 0 {
		fmt.Fprintln(w, "No duplicate files found.")
		return
	}

	for _, g := range groups {
		fmt.Fprintf(w, "Hash: %s | Size: %d bytes | Copies: %d | Wasted: %d bytes\n",
			g.Hash, g.Size, g.Count, g.WastedBytes)
		for _, f := range g.Files {
			fmt.Fprintf(w, "  - [%s] %s:%s (%s)\n", f.ScanID, f.Hostname, f.AbsPath, f.RelPath)
		}
	}
}
