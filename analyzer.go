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

// skipGenerated backs the -importalias.skip_generated flag (DEC-2.4). Per
// FR-5.7 generated files are skipped by default, so it defaults to true.
var skipGenerated = true

func init() {
	Analyzer.Flags.BoolVar(&skipGenerated, "skip_generated", true,
		"skip files carrying a generated-code marker (// Code generated ... DO NOT EDIT.)")
}

func run(pass *analysis.Pass) (any, error) {
	// pass.TypesInfo is already populated by the analysis driver, so no
	// separate go/packages load is needed here (that's the CLI's job, per
	// DEC-11.3, since a standalone run has no driver to do it for it).
	occs := scan.FromFiles(pass.Fset, pass.Files, pass.TypesInfo, scan.Options{
		Package:       pass.Pkg.Path(),
		SkipGenerated: skipGenerated,
	})
	decisions, collisions, duplicates := decide.Decide(occs, nil, decide.Options{})

	for _, d := range decisions {
		if d.Tie {
			continue
		}
		for _, o := range d.Inconsistent {
			msg := fmt.Sprintf("import %q should use alias %s, not %s (package-wide majority, fix target)",
				d.Path, aliasDisplay(d.WantAlias), aliasDisplay(o.Alias))
			// Report at every usage site (the actual fix targets), falling
			// back to the import declaration itself if none were resolved
			// (e.g. no type info available).
			if len(o.UsePos) == 0 {
				pass.Reportf(o.Pos, "%s", msg)
				continue
			}
			for _, pos := range o.UsePos {
				pass.Reportf(pos, "%s", msg)
			}
		}
	}
	for _, c := range collisions {
		if len(c.Occurrences) == 0 {
			continue // AliasCollision always has len>=2 Occurrences; zero-value defense only.
		}
		pass.Reportf(c.Occurrences[0].Pos, "alias %q is used for multiple import paths in this package", c.Alias)
	}
	for _, dup := range duplicates {
		for _, o := range dup.Occurrences {
			pass.Reportf(o.Pos, "import %q is imported multiple times in this file with different aliases", dup.Path)
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
