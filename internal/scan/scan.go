// Package scan collects import (path, alias) occurrences from parsed Go
// files, for both the go vet Analyzer and the standalone CLI to consume.
package scan

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"github.com/podhmo/go-importalias/internal/genfile"
	"github.com/podhmo/go-importalias/internal/shape"
)

// Options controls how FromFiles collects occurrences.
type Options struct {
	// Package is the import path recorded on every resulting Occurrence.
	Package string

	// SkipGenerated, when true, drops files carrying a generated-code marker
	// (// Code generated ... DO NOT EDIT., DEC-2.3) so their imports never
	// become Occurrences (FR-5.7). The default is false here; the caller
	// (analyzer/CLI) supplies the FR-5.7 "skip by default" policy.
	SkipGenerated bool

	// IncludeTests, when false, drops *_test.go files so their imports
	// never become Occurrences. The default is false here; the caller
	// (analyzer/CLI) supplies the tool-wide policy (DEC-11.23: both entry
	// points include test files by default, following go vet).
	IncludeTests bool
}

// FromFiles walks the import declarations of files directly via
// file.Imports, without go/ast/inspector.Inspector: a single package's
// import declarations are plain top-level data on *ast.File, so no
// traversal machinery is needed (see docs/02notice.md round 6).
//
// Blank imports (alias "_") and dot imports (alias ".") are excluded: they
// aren't part of the "which alias is correct" question this tool answers,
// and letting them leak into internal/decide's majority vote would corrupt
// it (DEC-11.19).
//
// typesInfo is optional (may be nil, in which case Occurrence.UsePos is left
// empty). When provided, it must describe files (Defs/Uses/Implicits
// populated by type-checking exactly these files as one package) — it is
// used to resolve, for each import, every qualified-identifier reference to
// it (the fix target locations, not just the import declaration itself).
func FromFiles(fset *token.FileSet, files []*ast.File, typesInfo *types.Info, opts Options) []shape.Occurrence {
	var occs []shape.Occurrence
	for _, file := range files {
		if opts.SkipGenerated && genfile.IsGenerated(file) {
			continue
		}
		filename := fset.Position(file.Pos()).Filename
		isTest := strings.HasSuffix(filename, "_test.go")
		if isTest && !opts.IncludeTests {
			continue
		}
		for _, imp := range file.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				continue
			}
			alias := ""
			if imp.Name != nil {
				alias = imp.Name.Name
			}
			if alias == "_" || alias == "." {
				continue
			}

			var usePos []token.Pos
			if pkgName := shape.PkgNameOf(typesInfo, imp); pkgName != nil {
				for _, ident := range shape.SelectorIdentsOf(file, typesInfo, pkgName) {
					usePos = append(usePos, ident.Pos())
				}
			}

			occs = append(occs, shape.Occurrence{
				Package: opts.Package,
				Path:    path,
				Alias:   alias,
				Pos:     imp.Pos(),
				IsTest:  isTest,
				File:    filename,
				UsePos:  usePos,
			})
		}
	}
	return occs
}
