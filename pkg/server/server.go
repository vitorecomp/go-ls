// Package server implements the HTTP REST API and serves the Next.js frontend for go-ls.
package server

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vitorecomp/go-ls/pkg/models"
	"github.com/vitorecomp/go-ls/pkg/store"
)

// Server exposes REST endpoints backed by a store.Store and serves the Next.js frontend
// (either from static export files or reverse-proxied from a live Next.js dev server).
type Server struct {
	store      store.Store
	uiDir      string
	uiProxyURL string
	mux        *http.ServeMux
}

// New constructs a Server using the default Next.js export directory ("./web/out") if present.
func New(st store.Store) *Server {
	return NewWithUI(st, "./web/out")
}

// NewWithUI constructs a Server that serves the compiled Next.js static files from uiDir
// (or proxies to GOLS_UI_PROXY if configured in development mode).
func NewWithUI(st store.Store, uiDir string) *Server {
	return NewWithUIAndProxy(st, uiDir, strings.TrimSpace(os.Getenv("GOLS_UI_PROXY")))
}

// NewWithUIAndProxy constructs a Server that proxies frontend requests to uiProxyURL when
// non-empty (enabling Next.js Fast Refresh / HMR in dev mode), or serves static files from uiDir.
func NewWithUIAndProxy(st store.Store, uiDir string, uiProxyURL string) *Server {
	s := &Server{
		store:      st,
		uiDir:      resolveUIDir(uiDir),
		uiProxyURL: strings.TrimSpace(uiProxyURL),
		mux:        http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/healthz" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /api/v1/computers", s.handleListComputers)
	s.mux.HandleFunc("POST /api/v1/scans", s.handleCreateScan)
	s.mux.HandleFunc("POST /api/v1/scans/batch", s.handleSaveScanBatch)
	s.mux.HandleFunc("POST /api/v1/scans/{id}/batch", s.handleSaveScanBatch)
	s.mux.HandleFunc("GET /api/v1/scans", s.handleListScans)
	s.mux.HandleFunc("GET /api/v1/scans/{id}", s.handleGetScan)
	s.mux.HandleFunc("GET /api/v1/scans/{id}/history", s.handleGetScanHistory)
	s.mux.HandleFunc("GET /api/v1/diff", s.handleDiffScans)
	s.mux.HandleFunc("GET /api/v1/duplicates", s.handleFindDuplicates)

	if s.uiProxyURL != "" {
		if target, err := url.Parse(s.uiProxyURL); err == nil && target.Host != "" {
			proxy := httputil.NewSingleHostReverseProxy(target)
			proxy.FlushInterval = -1
			proxy.Transport = &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   1500 * time.Millisecond,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   5 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			}
			proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, _ error) {
				if s.uiDir != "" {
					http.FileServer(http.Dir(s.uiDir)).ServeHTTP(w, req)
					return
				}
				s.handleFallbackIndex(w, req)
			}
			s.mux.Handle("/", proxy)
			return
		}
	}

	if s.uiDir != "" {
		fs := http.FileServer(http.Dir(s.uiDir))
		s.mux.Handle("GET /", fs)
	} else {
		s.mux.HandleFunc("GET /", s.handleFallbackIndex)
	}
}

func resolveUIDir(preferred string) string {
	candidates := []string{
		preferred,
		"./web/out",
	}
	if exePath, err := os.Executable(); err == nil && exePath != "" {
		if resolvedExe, err := filepath.EvalSymlinks(exePath); err == nil {
			exePath = resolvedExe
		}
		candidates = append(candidates, filepath.Join(filepath.Dir(exePath), "..", "share", "go-ls", "web"))
	}
	if xdgDataHome := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); xdgDataHome != "" {
		candidates = append(candidates, filepath.Join(xdgDataHome, "go-ls", "web"))
	}
	if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
		candidates = append(candidates, filepath.Join(homeDir, ".local", "share", "go-ls", "web"))
	}
	candidates = append(candidates,
		"/usr/local/share/go-ls/web",
		"/usr/share/go-ls/web",
	)
	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		info, err := os.Stat(filepath.Join(dir, "index.html"))
		if err == nil && !info.IsDir() {
			return dir
		}
	}
	return ""
}

func (s *Server) handleFallbackIndex(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>go-ls Server</title>
</head>
<body>
  <h1>go-ls API Server Online</h1>
  <p>Run <code>cd web && npm install && npm run build</code> to build the Next.js frontend bundle.</p>
</body>
</html>`))
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

const maxScanPayloadBytes = 64 << 20 // 64 MiB

func (s *Server) handleCreateScan(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxScanPayloadBytes)
	var payload models.ScanRecord
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	summary, err := s.store.SaveScan(r.Context(), payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, summary)
}

func (s *Server) handleSaveScanBatch(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxScanPayloadBytes)
	var payload models.ScanBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}
	if pathID := strings.TrimSpace(r.PathValue("id")); pathID != "" && strings.TrimSpace(payload.ID) == "" {
		payload.ID = pathID
	}

	summary, err := s.store.SaveScanBatch(r.Context(), payload)
	if err != nil {
		if errors.Is(err, store.ErrScanNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleListComputers(w http.ResponseWriter, r *http.Request) {
	computers, err := s.store.ListComputers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if computers == nil {
		computers = []models.Computer{}
	}
	writeJSON(w, http.StatusOK, computers)
}

func (s *Server) handleListScans(w http.ResponseWriter, r *http.Request) {
	summaries, err := s.store.ListScans(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if summaries == nil {
		summaries = []models.ScanSummary{}
	}
	writeJSON(w, http.StatusOK, summaries)
}

func (s *Server) handleGetScan(w http.ResponseWriter, r *http.Request) {
	scanID := r.PathValue("id")
	rec, err := s.store.GetScan(r.Context(), scanID)
	if err != nil {
		if errors.Is(err, store.ErrScanNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) handleGetScanHistory(w http.ResponseWriter, r *http.Request) {
	scanID := r.PathValue("id")
	history, err := s.store.GetScanHistory(r.Context(), scanID)
	if err != nil {
		if errors.Is(err, store.ErrScanNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if history == nil {
		history = []models.ScanHistoryEntry{}
	}
	writeJSON(w, http.StatusOK, history)
}

func (s *Server) handleDiffScans(w http.ResponseWriter, r *http.Request) {
	baseID := r.URL.Query().Get("base")
	targetID := r.URL.Query().Get("target")
	if baseID == "" || targetID == "" {
		writeError(w, http.StatusBadRequest, "both 'base' and 'target' query parameters are required")
		return
	}

	report, err := s.store.DiffScans(r.Context(), baseID, targetID)
	if err != nil {
		if errors.Is(err, store.ErrScanNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleFindDuplicates(w http.ResponseWriter, r *http.Request) {
	scanID := r.URL.Query().Get("scan_id")
	groups, err := s.store.FindDuplicates(r.Context(), scanID)
	if err != nil {
		if errors.Is(err, store.ErrScanNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if groups == nil {
		groups = []models.DuplicateGroup{}
	}
	writeJSON(w, http.StatusOK, groups)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
