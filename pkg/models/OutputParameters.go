package models

import (
	"github.com/vitorecomp/go-ls/pkg/cli"
)

type OutputParameters struct {
	Color      bool
	ShowHidden bool
	Hash       bool
	List       bool
	Comparable bool

	//new struture generate this two
	//separator
	//fields

	//trasform on a go template
	//go template
	//vertical
}

func BuildOutputParameters(arguments cli.CLI) OutputParameters {

	return OutputParameters{
		Color:      arguments.Color == "always" || arguments.Color == "auto",
		Hash:       arguments.Hash,
		ShowHidden: arguments.ShowHidden,
		List:       arguments.List,
		Comparable: arguments.Comparable,
	}
}
