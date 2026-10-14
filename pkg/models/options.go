// Package models defines the core domain types and configuration options for go-ls.
package models

// OutputChan is a channel that streams discovered File entries from the looker to the output stage.
type OutputChan chan File

// HashAlgorithm specifies the checksum algorithm used when hashing file contents.
type HashAlgorithm string

const (
	// HashCRC32 computes an IEEE CRC-32 checksum formatted as hexadecimal.
	HashCRC32 HashAlgorithm = "crc32"
	// HashSHA256 computes a SHA-256 cryptographic digest formatted as hexadecimal.
	HashSHA256 HashAlgorithm = "sha256"
)

// OutputFormat specifies how discovered File entries are rendered.
type OutputFormat string

const (
	// FormatDefault prints file and directory names in a compact console view.
	FormatDefault OutputFormat = "default"
	// FormatLong prints detailed file metadata (permissions, owner, group, size, timestamp, optional hash, name).
	FormatLong OutputFormat = "long"
	// FormatCompare prints CSV rows (hash, size, type, path) suitable for comparing directory trees.
	FormatCompare OutputFormat = "compare"
	// FormatJSON prints newline-delimited JSON objects (NDJSON) for each entry.
	FormatJSON OutputFormat = "json"
)

// LookOptions configures how the directory walker discovers and processes files.
type LookOptions struct {
	Recursive  bool
	ShowHidden bool
	Hash       bool
	HashAlgo   HashAlgorithm
	Workers    int
}

// LookParameters is an alias for LookOptions kept for backwards compatibility.
type LookParameters = LookOptions

// OutputOptions configures how the output controller formats and prints entries.
type OutputOptions struct {
	Format     OutputFormat
	ShowHidden bool
	Hash       bool
	NoColor    bool
}

// OutputParameters is an alias for OutputOptions kept for backwards compatibility.
type OutputParameters = OutputOptions
