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

	// File is the filename the import was scanned from, as resolved by the
	// scanner's *token.FileSet. It lets consumers regroup occurrences by
	// file without needing to carry a FileSet of their own (e.g. FR-6.16's
	// same-file duplicate-import detection in internal/decide).
	File string

	// UsePos holds the position of every qualified-identifier reference
	// (pkg.Symbol) that resolves to this import, within the same file. It is
	// only populated when the scanner was given type information; it is the
	// fix target list a diagnostic should point at, not just the import
	// declaration itself.
	UsePos []token.Pos
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

// DuplicateImport is one import path that is imported more than once within
// a single file under two or more different aliases (FR-6.16). This is a
// file-scoped axis, independent of FR-6.10's package-wide majority vote.
type DuplicateImport struct {
	Package     string
	File        string
	Path        string
	Occurrences []Occurrence // every occurrence of this path in the file, sorted by position
}
