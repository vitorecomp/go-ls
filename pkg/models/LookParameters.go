package models

import "github.com/vitorecomp/go-ls/pkg/cli"

type LookParameters struct {
	Recursive bool
	Hash      bool
}

func BuildLookParameters(arguments cli.CLI) LookParameters {
	return LookParameters{
		Recursive: arguments.Recursive,
		Hash:      arguments.Hash,
	}
}
