// Package genfile decides whether a parsed Go file is machine-generated, so
// that scan and the analyzer can leave it out of analysis and fixing. It is a
// pure function over the AST with no I/O (DEC-11.4).
package genfile

import (
	"go/ast"
	"regexp"
)

// generatedPattern is the standard generated-code marker (DEC-2.3,
// https://golang.org/s/generatedcode): a line comment whose entire text is
// "// Code generated <anything> DO NOT EDIT.".
var generatedPattern = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

// IsGenerated reports whether file carries a generated-code marker in its
// leading comment band — the comments that appear before the package clause,
// which is the first non-comment, non-blank line of the file (DEC-2.3).
//
// Build-constraint lines such as "//go:build linux" may sit alongside the
// marker in that band; they simply don't match the pattern and are skipped.
// A marker that appears only later (e.g. inside a function body, after the
// package clause) is not treated as a generated-file marker.
//
// file must have been parsed with parser.ParseComments for its comment band to
// be populated; otherwise this always returns false.
func IsGenerated(file *ast.File) bool {
	for _, group := range file.Comments {
		for _, c := range group.List {
			// Comment groups are ordered by position, so the first comment at
			// or past the package keyword marks the end of the leading band.
			if c.Pos() >= file.Package {
				return false
			}
			if generatedPattern.MatchString(c.Text) {
				return true
			}
		}
	}
	return false
}
