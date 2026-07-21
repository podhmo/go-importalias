// Package shape holds the domain types shared by scan, decide, and fix, plus
// the on-disk config file type and its I/O.
package shape

import "go/token"

// Occurrence is one observed import (path, alias) pair at a position.
// Alias is "" when the import has no explicit name.
type Occurrence struct {
	Package string
	Path    string
	Alias   string
	Pos     token.Pos
	IsTest  bool
}

// Decision is the outcome of deciding the "correct" alias for one
// (package, path) pair.
type Decision struct {
	Package      string
	Path         string
	WantAlias    string
	Tie          bool
	TieCandidate []string
	Inconsistent []Occurrence
}

// IsTie reports whether this decision could not be resolved automatically.
func (d Decision) IsTie() bool {
	return d.Tie
}

// AliasCollision is one alias used for more than one import path within a
// package.
type AliasCollision struct {
	Package     string
	Alias       string
	Occurrences []Occurrence
}
