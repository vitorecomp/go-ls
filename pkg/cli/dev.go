package cli

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

const devReloadDebounce = 250 * time.Millisecond

// runDevSupervisor watches Go source files using fsnotify, automatically rebuilds and restarts
// the go-ls server on changes, and optionally starts the Next.js dev server with HMR proxying.
func (s *ServeCmd) runDevSupervisor(ctx context.Context) error {
	tmpDir, err := os.MkdirTemp("", "go-ls-dev-*")
	if err != nil {
		return fmt.Errorf("creating temp dev directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	binPath := filepath.Join(tmpDir, "go-ls-dev-server")
	uiProxy := s.UIProxyURL

	var nextCmd *exec.Cmd
	if uiProxy == "" {
		if _, err := os.Stat(filepath.Join("web", "package.json")); err == nil {
			if _, lookErr := exec.LookPath("npm"); lookErr == nil {
				uiProxy = "http://127.0.0.1:3000"
				log.Printf("[dev] starting Next.js HMR dev server on %s...", uiProxy)
				nextCmd = exec.Command("npm", "run", "dev", "--", "--port", "3000")
				nextCmd.Dir = "web"
				nextCmd.Stdout = os.Stdout
				nextCmd.Stderr = os.Stderr
				nextCmd.Env = append(os.Environ(), "NODE_ENV=development")
				nextCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
				if err := nextCmd.Start(); err != nil {
					log.Printf("[dev] warning: could not start Next.js dev server: %v (falling back to static UI)", err)
					nextCmd = nil
					uiProxy = ""
				}
			}
		}
	}
	defer func() {
		killProcessGroup(nextCmd)
	}()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating dev file watcher: %w", err)
	}
	defer watcher.Close()

	addDevSourceWatches(watcher, ".")

	var serverCmd *exec.Cmd
	defer func() {
		killProcessGroup(serverCmd)
	}()

	buildAndStart := func() {
		log.Printf("[dev] compiling server binary...")
		buildCmd := exec.CommandContext(ctx, "go", "build", "-o", binPath, "./cmd/server")
		buildOut, buildErr := buildCmd.CombinedOutput()
		if buildErr != nil {
			log.Printf("[dev] build failed (waiting for next file save):\n%s", strings.TrimSpace(string(buildOut)))
			return
		}

		killProcessGroup(serverCmd)

		args := []string{
			"--listen=" + s.ListenAddr,
			"--db-url=" + s.DBURL,
			"--ui-dir=" + s.UIDir,
		}
		if uiProxy != "" {
			args = append(args, "--ui-proxy="+uiProxy)
		}

		cmd := exec.Command(binPath, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = append(os.Environ(), "GOLS_DEV_CHILD=1")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		if err := cmd.Start(); err != nil {
			log.Printf("[dev] failed to start server: %v", err)
			return
		}
		serverCmd = cmd
		log.Printf("[dev] server running (PID %d) — watching .go files for hot reload", cmd.Process.Pid)
	}

	buildAndStart()

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
			log.Printf("[dev] shutting down hot-reload supervisor...")
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Create) {
				if info, statErr := os.Stat(event.Name); statErr == nil && info.IsDir() {
					addDevSourceWatches(watcher, event.Name)
				}
			}
			if !shouldReloadGoFile(event.Name) {
				continue
			}

			if debounceTimer == nil {
				debounceTimer = time.NewTimer(devReloadDebounce)
			} else {
				if !debounceTimer.Stop() {
					select {
					case <-debounceTimer.C:
					default:
					}
				}
				debounceTimer.Reset(devReloadDebounce)
			}
			debounceChan = debounceTimer.C

		case watchErr, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("[dev] watcher error: %v", watchErr)

		case <-debounceChan:
			debounceChan = nil
			log.Printf("[dev] Go source change detected, reloading server...")
			buildAndStart()
		}
	}
}

func addDevSourceWatches(watcher *fsnotify.Watcher, root string) {
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		name := d.Name()
		if path != "." && (strings.HasPrefix(name, ".") || name == "web" || name == "documents" || name == "scripts" || name == "script" || name == "vendor" || name == "node_modules") {
			return filepath.SkipDir
		}
		_ = watcher.Add(path)
		return nil
	})
}

func shouldReloadGoFile(path string) bool {
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".") || strings.HasSuffix(base, "_test.go") {
		return false
	}
	if strings.HasSuffix(base, ".go") || base == "go.mod" || base == "go.sum" {
		return true
	}
	return false
}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil && pgid > 0 {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
	} else {
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		if pgid > 0 {
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		} else {
			_ = cmd.Process.Kill()
		}
		<-done
	}
}
