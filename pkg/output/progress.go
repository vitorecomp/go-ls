package output

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/vitorecomp/go-ls/pkg/models"
)

const (
	progressBarWidth       = 24
	progressRedrawInterval = 35 * time.Millisecond
)

// ProgressBar renders a synchronized terminal progress bar to an error/status writer
// while coordinating with stdout/stderr writes so streaming file listings never garble.
type ProgressBar struct {
	mu         sync.Mutex
	w          io.Writer
	noColor    bool
	startTime  time.Time
	lastDraw   time.Time
	lastSnap   models.ProgressSnapshot
	barVisible bool
	midLine    bool
	finished   bool
}

// NewProgressBar creates a new ProgressBar writing to w.
func NewProgressBar(w io.Writer, noColor bool) *ProgressBar {
	if w == nil {
		w = io.Discard
	}
	return &ProgressBar{
		w:         w,
		noColor:   noColor,
		startTime: time.Now(),
	}
}

// Update records a new ProgressSnapshot from the directory walker and redraws the bar
// when appropriate.
func (p *ProgressBar) Update(snap models.ProgressSnapshot) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.finished {
		return
	}

	p.lastSnap = snap
	if p.midLine {
		return
	}

	now := time.Now()
	if !snap.Done && !p.lastDraw.IsZero() && now.Sub(p.lastDraw) < progressRedrawInterval {
		return
	}
	p.drawLocked(now, false)
}

// Finish renders the completed 100% progress bar on its own line.
func (p *ProgressBar) Finish() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.finished {
		return
	}
	p.finished = true
	p.lastSnap.Done = true
	if p.lastSnap.Completed > p.lastSnap.Discovered {
		p.lastSnap.Discovered = p.lastSnap.Completed
	}
	p.drawLocked(time.Now(), true)
}

// WrapStdout returns an io.Writer that clears the active progress bar before writing
// to out and redraws the progress bar once a newline is written.
func (p *ProgressBar) WrapStdout(out io.Writer) io.Writer {
	if p == nil || out == nil {
		return out
	}
	return &progressCoordWriter{
		pb:       p,
		target:   out,
		isStdout: true,
	}
}

// WrapStderr returns an io.Writer that clears the active progress bar before writing
// diagnostic/error messages and redraws the progress bar afterward.
func (p *ProgressBar) WrapStderr(errOut io.Writer) io.Writer {
	if p == nil || errOut == nil {
		return errOut
	}
	return &progressCoordWriter{
		pb:       p,
		target:   errOut,
		isStdout: false,
	}
}

type progressCoordWriter struct {
	pb       *ProgressBar
	target   io.Writer
	isStdout bool
}

func (w *progressCoordWriter) Write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	w.pb.mu.Lock()
	defer w.pb.mu.Unlock()

	if w.pb.barVisible {
		_, _ = io.WriteString(w.pb.w, "\r\033[K")
		w.pb.barVisible = false
	}

	n, err := w.target.Write(b)
	if w.isStdout {
		w.pb.midLine = b[len(b)-1] != '\n'
	}

	if !w.pb.finished && !w.pb.midLine {
		w.pb.drawLocked(time.Now(), false)
	}
	return n, err
}

func (p *ProgressBar) drawLocked(now time.Time, finalNewline bool) {
	elapsed := now.Sub(p.startTime)
	line := FormatProgressBar(p.lastSnap, elapsed, p.noColor)
	if finalNewline {
		_, _ = fmt.Fprintf(p.w, "\r\033[K%s\n", line)
		p.barVisible = false
		return
	}
	_, _ = fmt.Fprintf(p.w, "\r\033[K%s", line)
	p.barVisible = true
	p.lastDraw = now
}

// FormatProgressBar formats a single-line progress bar string for snap.
func FormatProgressBar(snap models.ProgressSnapshot, elapsed time.Duration, noColor bool) string {
	disc := snap.Discovered
	comp := snap.Completed
	if comp > disc {
		disc = comp
	}

	pct := 0
	switch {
	case snap.Done:
		pct = 100
		comp = disc
	case disc > 0:
		pct = int((comp * 100) / disc)
		if pct > 99 {
			pct = 99
		}
	}

	filled := (pct * progressBarWidth) / 100
	if filled > progressBarWidth {
		filled = progressBarWidth
	}
	empty := progressBarWidth - filled

	barStr := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	if !noColor {
		c := color.New(color.FgCyan)
		if snap.Done {
			c = color.New(color.FgGreen, color.Bold)
		}
		c.EnableColor()
		barStr = c.Sprint(barStr)
	}

	summary := fmt.Sprintf("[%s] %3d%% | %d/%d items | %d dirs | %s | %s",
		barStr,
		pct,
		comp,
		disc,
		snap.Dirs,
		FormatBytes(snap.Bytes),
		formatElapsed(elapsed),
	)

	if !snap.Done && snap.Current != "" {
		base := filepath.Base(snap.Current)
		if base != "" && base != "." && base != "/" {
			if len(base) > 28 {
				base = base[:25] + "..."
			}
			summary += " | " + base
		}
	}
	return summary
}

// FormatBytes formats a byte count into a compact human-readable string.
func FormatBytes(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	}
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	val := float64(b)
	unitIdx := -1
	for val >= 1024 && unitIdx < len(units)-1 {
		val /= 1024
		unitIdx++
	}
	return fmt.Sprintf("%.1f %s", val, units[unitIdx])
}

func formatElapsed(d time.Duration) string {
	if d < time.Millisecond {
		return "<1ms"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
