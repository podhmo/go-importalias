// Package scan collects import (path, alias) occurrences from parsed Go
// files, for both the go vet Analyzer and the standalone CLI to consume.
package scan

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/podhmo/go-importalias/internal/shape"
)

// Options controls how FromFiles collects occurrences.
type Options struct {
	// Package is the import path recorded on every resulting Occurrence.
	Package string
}

// FromFiles walks the import declarations of files directly via
// file.Imports, without go/ast/inspector.Inspector: a single package's
// import declarations are plain top-level data on *ast.File, so no
// traversal machinery is needed (see docs/02notice.md round 6).
func FromFiles(fset *token.FileSet, files []*ast.File, opts Options) []shape.Occurrence {
	var occs []shape.Occurrence
	for _, file := range files {
		isTest := strings.HasSuffix(fset.Position(file.Pos()).Filename, "_test.go")
		for _, imp := range file.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				continue
			}
			alias := ""
			if imp.Name != nil {
				alias = imp.Name.Name
			}
			occs = append(occs, shape.Occurrence{
				Package: opts.Package,
				Path:    path,
				Alias:   alias,
				Pos:     imp.Pos(),
				IsTest:  isTest,
			})
		}
	}
	return occs
}
