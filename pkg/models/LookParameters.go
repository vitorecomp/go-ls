package models

import "github.com/vitorecomp/go-ls/pkg/cli"

type LookParameters struct {
	Recursive  bool
	ShowHidden bool
	Hash       bool
}

func BuildLookParameters(arguments cli.CLI) LookParameters {
	lookParams := LookParameters{
		Recursive:  arguments.Recursive,
		ShowHidden: arguments.ShowHidden,
		Hash:       arguments.Hash,
	}

	return lookParams
}
