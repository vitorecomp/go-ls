package cli

import (
	"runtime"

	"github.com/vitorecomp/go-ls/pkg/models"
)

// CLI defines the command-line flags and positional arguments parsed by Kong.
type CLI struct {
	ShowHidden bool   `help:"Show hidden files (entries starting with '.')." name:"all" short:"a"`
	Long       bool   `help:"Use a long listing format." name:"long" short:"l"`
	Recursive  bool   `help:"List subdirectories recursively." name:"recursive" short:"R"`
	Hash       bool   `help:"Compute checksums for every file and folder (combined hash of its contents)." name:"hash"`
	HashAlgo   string `help:"Checksum algorithm to use when hashing files (crc32, sha256)." enum:"crc32,sha256" default:"crc32" name:"hash-algo"`
	Compare    bool   `help:"Output CSV records (hash,size,type,path) for comparing directory trees (implies --hash)." name:"compare"`
	JSON       bool   `help:"Output newline-delimited JSON records for each entry." name:"json"`
	Color      string `help:"Colorize the output (auto, always, never)." enum:"auto,always,never" default:"auto" name:"color"`
	Workers    int    `help:"Number of concurrent workers for directory traversal and hashing (0 uses CPU count)." default:"0" name:"workers"`
	Version    bool   `help:"Output version information and exit." name:"version"`

	Paths []string `arg:"" optional:"" name:"path" help:"Files or directories to list." type:"path" default:"."`
}

// ToLookOptions converts parsed CLI flags into models.LookOptions.
func (c CLI) ToLookOptions() models.LookOptions {
	workers := c.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
		if workers < 4 {
			workers = 4
		}
	}

	algo := models.HashCRC32
	if c.HashAlgo == string(models.HashSHA256) {
		algo = models.HashSHA256
	}

	return models.LookOptions{
		Recursive:  c.Recursive,
		ShowHidden: c.ShowHidden,
		Hash:       c.Hash || c.Compare,
		HashAlgo:   algo,
		Workers:    workers,
	}
}

// ToOutputOptions converts parsed CLI flags into models.OutputOptions.
func (c CLI) ToOutputOptions() models.OutputOptions {
	format := models.FormatDefault
	switch {
	case c.JSON:
		format = models.FormatJSON
	case c.Compare:
		format = models.FormatCompare
	case c.Long:
		format = models.FormatLong
	}

	return models.OutputOptions{
		Format:     format,
		ShowHidden: c.ShowHidden,
		Hash:       c.Hash || c.Compare,
		NoColor:    c.Color == "never",
	}
}
