package cli

import (
	"runtime"

	"github.com/vitorecomp/go-ls/pkg/models"
)

// DefaultServerURL is the default endpoint used by client commands when --server is not specified.
const DefaultServerURL = "http://localhost:8080"

// ScanCmd scans local files/directories and optionally uploads the inventory to the go-ls server.
type ScanCmd struct {
	ShowHidden bool   `help:"Show hidden files (entries starting with '.')." name:"all" short:"a"`
	Long       bool   `help:"Use a long listing format." name:"long" short:"l"`
	Recursive  bool   `help:"List subdirectories recursively." name:"recursive" short:"R"`
	Hash       bool   `help:"Compute checksums for every file and folder (combined hash of its contents)." name:"hash"`
	HashAlgo   string `help:"Checksum algorithm to use when hashing files (crc32, sha256)." enum:"crc32,sha256" default:"crc32" name:"hash-algo"`
	Compare    bool   `help:"Output CSV records (hash,size,type,path) for comparing directory trees (implies --hash)." name:"compare"`
	JSON       bool   `help:"Output newline-delimited JSON records for each entry." name:"json"`
	Color      string `help:"Colorize the output (auto, always, never)." enum:"auto,always,never" default:"auto" name:"color"`
	Workers    int    `help:"Number of concurrent workers for directory traversal and hashing (0 uses CPU count)." default:"0" name:"workers"`
	Live       bool   `help:"Keep monitoring the directory for changes using filesystem events (implies --hash)." name:"live"`

	ServerURL string `help:"Upload the completed scan to a go-ls server URL (e.g. http://localhost:8080)." env:"GOLS_SERVER_URL" name:"server"`
	ScanID    string `help:"Optional custom ID for the uploaded scan (auto-generated if omitted)." name:"scan-id"`
	Tag       string `help:"Optional label/tag to attach to the uploaded scan." name:"tag"`
	Quiet     bool   `help:"Suppress local file listing output when uploading to the server." name:"quiet" short:"q"`

	Paths []string `arg:"" optional:"" name:"path" help:"Files or directories to scan." type:"path" default:"."`
}

// ToLookOptions converts ScanCmd flags into models.LookOptions.
func (s ScanCmd) ToLookOptions() models.LookOptions {
	workers := s.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
		if workers < 4 {
			workers = 4
		}
	}

	algo := models.HashCRC32
	if s.HashAlgo == string(models.HashSHA256) {
		algo = models.HashSHA256
	}

	// Uploading to the server, running --compare, or monitoring with --live automatically enables hashing.
	enableHash := s.Hash || s.Compare || s.ServerURL != "" || s.Live

	return models.LookOptions{
		Recursive:  s.Recursive,
		ShowHidden: s.ShowHidden,
		Hash:       enableHash,
		HashAlgo:   algo,
		Workers:    workers,
		Live:       s.Live,
	}
}

// ToOutputOptions converts ScanCmd flags into models.OutputOptions.
func (s ScanCmd) ToOutputOptions() models.OutputOptions {
	format := models.FormatDefault
	switch {
	case s.JSON:
		format = models.FormatJSON
	case s.Compare:
		format = models.FormatCompare
	case s.Long:
		format = models.FormatLong
	}

	return models.OutputOptions{
		Format:     format,
		ShowHidden: s.ShowHidden,
		Hash:       s.Hash || s.Compare || s.Live || s.ServerURL != "",
		NoColor:    s.Color == "never",
	}
}

// ScansCmd lists or inspects scans stored on the go-ls server.
type ScansCmd struct {
	ServerURL string `help:"go-ls server URL." default:"http://localhost:8080" env:"GOLS_SERVER_URL" name:"server"`
	JSON      bool   `help:"Output results as JSON." name:"json"`
	ScanID    string `arg:"" optional:"" name:"scan-id" help:"Optional scan ID to inspect in detail."`
}

// DiffCmd compares two stored scans on the go-ls server.
type DiffCmd struct {
	ServerURL string `help:"go-ls server URL." default:"http://localhost:8080" env:"GOLS_SERVER_URL" name:"server"`
	JSON      bool   `help:"Output diff report as JSON." name:"json"`
	Color     string `help:"Colorize the output (auto, always, never)." enum:"auto,always,never" default:"auto" name:"color"`
	BaseID    string `arg:"" name:"base-scan-id" help:"Base scan ID."`
	TargetID  string `arg:"" name:"target-scan-id" help:"Target scan ID to compare against base."`
}

// DuplicatesCmd finds duplicate files by content hash on the go-ls server.
type DuplicatesCmd struct {
	ServerURL string `help:"go-ls server URL." default:"http://localhost:8080" env:"GOLS_SERVER_URL" name:"server"`
	JSON      bool   `help:"Output duplicate groups as JSON." name:"json"`
	ScanID    string `arg:"" optional:"" name:"scan-id" help:"Optional scan ID to restrict duplicate search (searches across all scans if omitted)."`
}

// ServeCmd runs the go-ls HTTP API and Next.js frontend server.
type ServeCmd struct {
	ListenAddr string `help:"TCP address for the HTTP server to listen on." default:":8080" env:"GOLS_LISTEN_ADDR" name:"listen"`
	DBURL      string `help:"Database connection URL (postgres://... or memory://)." default:"memory://" env:"GOLS_DB_URL" name:"db-url"`
	UIDir      string `help:"Directory containing the compiled Next.js static export." default:"./web/out" env:"GOLS_UI_DIR" name:"ui-dir"`
	UIProxyURL string `help:"Optional Next.js dev server URL to reverse-proxy UI & HMR requests to." env:"GOLS_UI_PROXY" name:"ui-proxy"`
	Dev        bool   `help:"Enable hot-reload development mode (auto-rebuilds Go server on .go changes and proxies Next.js HMR)." env:"GOLS_DEV" name:"dev"`
}

// CLI is the root command-line structure supporting both client and server subcommands.
type CLI struct {
	Version bool `help:"Output version information and exit." name:"version"`

	Scan       ScanCmd       `cmd:"" help:"Scan local directories/files and optionally upload results to the server."`
	Scans      ScansCmd      `cmd:"" help:"List stored scans on the server, or inspect a single scan by ID."`
	Diff       DiffCmd       `cmd:"" help:"Compare two scans stored on the server (added, deleted, modified entries)."`
	Duplicates DuplicatesCmd `cmd:"" help:"Find duplicate files by hash within a scan or across all stored scans."`
	Serve      ServeCmd      `cmd:"" help:"Run the go-ls HTTP server."`
}

// NormalizeArgs prepends the default "scan" subcommand when the user invokes go-ls
// directly with flags or paths (e.g., `go-ls -la --hash` or `go-ls ./pkg`).
func NormalizeArgs(args []string) []string {
	if len(args) == 0 {
		return []string{"scan"}
	}
	switch args[0] {
	case "scan", "scans", "diff", "duplicates", "serve", "-h", "--help", "help", "--version":
		return args
	default:
		return append([]string{"scan"}, args...)
	}
}
