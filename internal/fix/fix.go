// Package fix rewrites a file's import clause and matching qualified
// identifiers to the alias decided by internal/decide.
//
// This experiment handles the "rename an existing alias to the decided
// majority alias" case, FR-6.16 duplicate imports, and the safe FR-6.11
// collision case where one colliding path can be unaliased. It implements
// FR-7.21's collision check (skip the rewrite if the resulting identifier
// would collide with something already in scope, e.g. a local variable); see
// docs/02notice.md round 8.
package fix

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"strconv"

	"github.com/podhmo/go-importalias/internal/shape"
	"golang.org/x/tools/go/ast/astutil"
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
	return ApplyToFileWithCollisionsAndDuplicates(fset, file, typesInfo, decisions, nil, nil)
}

// ApplyToFileWithDuplicates is ApplyToFile plus FR-6.16 handling: within one
// file, duplicate imports of the same path under different aliases are merged
// into the canonical alias decided by internal/decide, and surplus import
// specs are removed.
func ApplyToFileWithDuplicates(fset *token.FileSet, file *ast.File, typesInfo *types.Info, decisions []shape.Decision, duplicates []shape.DuplicateImport) (src []byte, changed bool, err error) {
	return ApplyToFileWithCollisionsAndDuplicates(fset, file, typesInfo, decisions, nil, duplicates)
}

// ApplyToFileWithCollisions is ApplyToFile plus FR-6.11 handling: for one
// explicit alias used by multiple import paths, keep the first path reported
// by internal/decide and unalias the remaining paths when that is safe.
func ApplyToFileWithCollisions(fset *token.FileSet, file *ast.File, typesInfo *types.Info, decisions []shape.Decision, collisions []shape.AliasCollision) (src []byte, changed bool, err error) {
	return ApplyToFileWithCollisionsAndDuplicates(fset, file, typesInfo, decisions, collisions, nil)
}

// ApplyToFileWithCollisionsAndDuplicates applies all currently supported fix
// axes in one pass.
func ApplyToFileWithCollisionsAndDuplicates(fset *token.FileSet, file *ast.File, typesInfo *types.Info, decisions []shape.Decision, collisions []shape.AliasCollision, duplicates []shape.DuplicateImport) (src []byte, changed bool, err error) {
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
	if applyAliasCollisions(fset, file, typesInfo, collisions) {
		changed = true
	}
	if applyDuplicateImports(fset, file, typesInfo, decisions, duplicates) {
		changed = true
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

func applyAliasCollisions(fset *token.FileSet, file *ast.File, typesInfo *types.Info, collisions []shape.AliasCollision) bool {
	changed := false
	for _, c := range collisions {
		if len(c.Occurrences) <= 1 {
			continue
		}
		for _, occ := range c.Occurrences[1:] {
			spec := findImportSpecAt(file, occ.Pos)
			if spec == nil || importAlias(spec) != c.Alias {
				continue
			}
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil || path != occ.Path {
				continue
			}
			pkgName := shape.PkgNameOf(typesInfo, spec)
			if pkgName == nil {
				continue
			}
			resolvedName := pkgName.Imported().Name()
			if resolvedName == c.Alias || collides(typesInfo, file, spec, pkgName, resolvedName) {
				continue
			}
			if !replaceImport(fset, file, c.Alias, "", occ.Path) {
				continue
			}
			for _, ident := range shape.SelectorIdentsOf(file, typesInfo, pkgName) {
				ident.Name = resolvedName
			}
			changed = true
		}
	}
	return changed
}

func applyDuplicateImports(fset *token.FileSet, file *ast.File, typesInfo *types.Info, decisions []shape.Decision, duplicates []shape.DuplicateImport) bool {
	changed := false
	for _, dup := range duplicates {
		d, ok := findDecision(decisions, dup.Package, dup.Path)
		if !ok || d.Tie {
			continue
		}
		if mergeDuplicateImport(fset, file, typesInfo, dup, d.WantAlias) {
			changed = true
		}
	}
	return changed
}

func findDecision(decisions []shape.Decision, pkg, path string) (shape.Decision, bool) {
	for _, d := range decisions {
		if d.Package == pkg && d.Path == path {
			return d, true
		}
	}
	return shape.Decision{}, false
}

func mergeDuplicateImport(fset *token.FileSet, file *ast.File, typesInfo *types.Info, dup shape.DuplicateImport, wantAlias string) bool {
	type importBinding struct {
		spec    *ast.ImportSpec
		pkgName *types.PkgName
	}
	bindings := make([]importBinding, 0, len(dup.Occurrences))
	for _, occ := range dup.Occurrences {
		spec := findImportSpecAt(file, occ.Pos)
		if spec == nil {
			continue
		}
		pkgName := shape.PkgNameOf(typesInfo, spec)
		if pkgName == nil {
			return false
		}
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil || path != dup.Path {
			return false
		}
		bindings = append(bindings, importBinding{spec: spec, pkgName: pkgName})
	}
	if len(bindings) <= 1 {
		return false
	}

	keep := -1
	for i, b := range bindings {
		if importAlias(b.spec) == wantAlias {
			keep = i
			break
		}
	}
	if keep == -1 {
		keep = 0
	}

	keepBinding := bindings[keep]
	resolvedName := wantAlias
	if resolvedName == "" {
		resolvedName = keepBinding.pkgName.Imported().Name()
	}

	if importAlias(keepBinding.spec) != wantAlias && collides(typesInfo, file, keepBinding.spec, keepBinding.pkgName, resolvedName) {
		return false
	}
	for i, b := range bindings {
		if i == keep {
			continue
		}
		if collidesWithCanonical(typesInfo, file, b.spec, b.pkgName, resolvedName, keepBinding.pkgName) {
			return false
		}
	}

	if importAlias(keepBinding.spec) != wantAlias {
		renameImportSpec(typesInfo, file, keepBinding.pkgName, keepBinding.spec, wantAlias, resolvedName)
	}
	for i, b := range bindings {
		if i == keep {
			continue
		}
		for _, ident := range shape.SelectorIdentsOf(file, typesInfo, b.pkgName) {
			ident.Name = resolvedName
		}
		astutil.DeleteNamedImport(fset, file, importAlias(b.spec), dup.Path)
	}
	return true
}

func importAlias(spec *ast.ImportSpec) string {
	if spec.Name == nil {
		return ""
	}
	return spec.Name.Name
}

func replaceImport(fset *token.FileSet, file *ast.File, oldAlias, newAlias, path string) bool {
	added := astutil.AddNamedImport(fset, file, newAlias, path)
	if !astutil.DeleteNamedImport(fset, file, oldAlias, path) {
		if added {
			astutil.DeleteNamedImport(fset, file, newAlias, path)
		}
		return false
	}
	return true
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

func collidesWithCanonical(typesInfo *types.Info, file *ast.File, spec *ast.ImportSpec, pkgName *types.PkgName, resolvedName string, canonical *types.PkgName) bool {
	if shape.NameVisibleAt(typesInfo, file, spec.Pos(), resolvedName, canonical) {
		return true
	}
	for _, ident := range shape.SelectorIdentsOf(file, typesInfo, pkgName) {
		if shape.NameVisibleAt(typesInfo, file, ident.Pos(), resolvedName, canonical) {
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
