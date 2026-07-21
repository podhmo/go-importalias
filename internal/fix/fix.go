// Package fix rewrites a file's import clause and matching qualified
// identifiers to the alias decided by internal/decide.
//
// This experiment only handles the "rename an existing alias to the decided
// majority alias" case (no astutil-style add/delete of import specs). It
// does implement FR-7.21's collision check (skip the rewrite if the
// resulting identifier would collide with something already in scope,
// e.g. a local variable) for this rename case; see docs/02notice.md round 8.
package fix

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"

	"github.com/podhmo/go-importalias/internal/shape"
)

// ApplyToFile rewrites file in place according to decisions, then formats it
// back into src. typesInfo must have been produced by type-checking file
// (Defs/Uses/Implicits/Scopes populated). changed is false (and src nil) if
// no decision applied to any import in this file — either because none
// matched, or because every match was skipped due to an identifier
// collision (the "trivial transformation" rule: never rewrite if it would
// collide with a local variable, another import, a top-level declaration,
// etc.).
func ApplyToFile(fset *token.FileSet, file *ast.File, typesInfo *types.Info, decisions []shape.Decision) (src []byte, changed bool, err error) {
	for _, d := range decisions {
		if d.Tie {
			continue
		}
		for _, occ := range d.Inconsistent {
			spec := findImportSpecAt(file, occ.Pos)
			if spec == nil {
				continue // this occurrence belongs to a different file
			}
			pkgName := shape.PkgNameOf(typesInfo, spec)
			if pkgName == nil {
				continue // no type info resolved for this import; can't safely rewrite
			}

			// WantAlias == "" means "no explicit alias"; the qualifier that
			// then appears in code is the package's own declared name (which
			// may differ from the import path's last segment), not "".
			resolvedName := d.WantAlias
			if resolvedName == "" {
				resolvedName = pkgName.Imported().Name()
			}

			if collides(typesInfo, file, spec, pkgName, resolvedName) {
				continue
			}

			renameImportSpec(typesInfo, file, pkgName, spec, d.WantAlias, resolvedName)
			changed = true
		}
	}

	if !changed {
		return nil, false, nil
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, false, err
	}
	return buf.Bytes(), true, nil
}

func findImportSpecAt(file *ast.File, pos token.Pos) *ast.ImportSpec {
	for _, imp := range file.Imports {
		if imp.Pos() == pos {
			return imp
		}
	}
	return nil
}

// collides reports whether renaming spec's qualifier to resolvedName would
// clash with some other identifier already visible at any of pkgName's use
// sites, including the import declaration itself — a local variable,
// another import, a top-level declaration, etc. Pre-existing shadowing that
// doesn't involve resolvedName (e.g. an unrelated local variable in a
// different function that never references this import) is not a collision
// and does not block the rewrite.
func collides(typesInfo *types.Info, file *ast.File, spec *ast.ImportSpec, pkgName *types.PkgName, resolvedName string) bool {
	if shape.NameVisibleAt(typesInfo, file, spec.Pos(), resolvedName, pkgName) {
		return true
	}
	for _, ident := range shape.SelectorIdentsOf(file, typesInfo, pkgName) {
		if shape.NameVisibleAt(typesInfo, file, ident.Pos(), resolvedName, pkgName) {
			return true
		}
	}
	return false
}

// renameImportSpec rewrites every qualified-identifier reference to pkgName
// (found via shape.SelectorIdentsOf) to resolvedName, then updates
// spec.Name: wantAlias == "" clears it (unaliased import, relying on the
// package's declared name — which is what resolvedName already is in that
// case), otherwise it's set to wantAlias explicitly. This mutates the
// existing ImportSpec node directly instead of going through
// astutil.AddNamedImport/DeleteNamedImport, since a plain rename doesn't
// need to add or remove an import declaration.
func renameImportSpec(typesInfo *types.Info, file *ast.File, pkgName *types.PkgName, spec *ast.ImportSpec, wantAlias, resolvedName string) {
	for _, ident := range shape.SelectorIdentsOf(file, typesInfo, pkgName) {
		ident.Name = resolvedName
	}

	if wantAlias == "" {
		spec.Name = nil
	} else {
		spec.Name = ast.NewIdent(wantAlias)
	}
}
