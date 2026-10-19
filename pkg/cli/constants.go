// Package cli defines command-line flags, constants, and option builders for go-ls.
package cli

import "github.com/vitorecomp/go-ls/pkg/models"

const (
	// Version is the semantic version of go-ls.
	Version = "0.2.0"

	// DefaultDateFormat is the timestamp layout used in long (-l) console output.
	DefaultDateFormat = models.DefaultDateFormat
)

type constants struct {
	DateFormat string
}

// Constants holds shared formatting constants for CLI output.
var Constants = constants{
	DateFormat: DefaultDateFormat,
}
