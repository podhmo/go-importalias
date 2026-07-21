// Package importalias exports the go vet Analyzer that reports inconsistent
// import path <-> alias mappings within a package.
package importalias

import (
	"fmt"

	"golang.org/x/tools/go/analysis"

	"github.com/podhmo/go-importalias/internal/decide"
	"github.com/podhmo/go-importalias/internal/scan"
)

// Analyzer reports, for each package, import paths that are aliased
// inconsistently across the package's files. It never writes to the
// filesystem.
var Analyzer = &analysis.Analyzer{
	Name: "importalias",
	Doc:  "reports inconsistent import path <-> alias mappings within a package",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	occs := scan.FromFiles(pass.Fset, pass.Files, scan.Options{Package: pass.Pkg.Path()})
	decisions := decide.Decide(occs, nil, decide.Options{})

	for _, d := range decisions {
		if d.Tie {
			continue
		}
		for _, o := range d.Inconsistent {
			pass.Reportf(o.Pos, "import %q should use alias %s, not %s (package-wide majority)",
				d.Path, aliasDisplay(d.WantAlias), aliasDisplay(o.Alias))
		}
	}
	return nil, nil
}

func aliasDisplay(alias string) string {
	if alias == "" {
		return "no alias"
	}
	return fmt.Sprintf("%q", alias)
}
