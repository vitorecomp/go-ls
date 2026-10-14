package cli

import (
	"testing"

	"github.com/alecthomas/kong"
	"github.com/vitorecomp/go-ls/pkg/models"
)

func TestParseAndOptionsConversion(t *testing.T) {
	var parsed CLI
	parser, err := kong.New(&parsed)
	if err != nil {
		t.Fatalf("kong.New failed: %v", err)
	}

	_, err = parser.Parse([]string{"-la", "-R", "--hash", "--hash-algo=sha256", "--color=never", "--workers=8", "/tmp"})
	if err != nil {
		t.Fatalf("parser.Parse failed: %v", err)
	}

	lookOpts := parsed.ToLookOptions()
	if !lookOpts.ShowHidden || !lookOpts.Recursive || !lookOpts.Hash {
		t.Errorf("unexpected LookOptions booleans: %+v", lookOpts)
	}
	if lookOpts.HashAlgo != models.HashSHA256 {
		t.Errorf("HashAlgo = %q, want %q", lookOpts.HashAlgo, models.HashSHA256)
	}
	if lookOpts.Workers != 8 {
		t.Errorf("Workers = %d, want 8", lookOpts.Workers)
	}

	outOpts := parsed.ToOutputOptions()
	if outOpts.Format != models.FormatLong {
		t.Errorf("Format = %q, want %q", outOpts.Format, models.FormatLong)
	}
	if !outOpts.NoColor {
		t.Errorf("expected NoColor=true when --color=never")
	}
	if len(parsed.Paths) != 1 || parsed.Paths[0] != "/tmp" {
		t.Errorf("Paths = %v, want [/tmp]", parsed.Paths)
	}
}

func TestCompareImpliesHashAndCompareFormat(t *testing.T) {
	var parsed CLI
	parser, err := kong.New(&parsed)
	if err != nil {
		t.Fatalf("kong.New failed: %v", err)
	}

	_, err = parser.Parse([]string{"--compare"})
	if err != nil {
		t.Fatalf("parser.Parse failed: %v", err)
	}

	lookOpts := parsed.ToLookOptions()
	if !lookOpts.Hash {
		t.Errorf("expected --compare to imply LookOptions.Hash=true")
	}

	outOpts := parsed.ToOutputOptions()
	if outOpts.Format != models.FormatCompare {
		t.Errorf("Format = %q, want %q", outOpts.Format, models.FormatCompare)
	}
}
