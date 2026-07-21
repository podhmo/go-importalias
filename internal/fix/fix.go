// Package fix rewrites a file's import clause and matching qualified
// identifiers to the alias decided by internal/decide.
//
// This experiment only handles the "rename an existing alias to the decided
// majority alias" case (no astutil-style add/delete of import specs, no
// FR-7.21 alias-removal collision check). See docs/02notice.md round 6.
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
// (Defs/Uses/Implicits populated). changed is false (and src nil) if no
// decision applied to any import in this file.
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
			renameImportSpec(typesInfo, file, spec, d.WantAlias)
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

// renameImportSpec resolves the *types.PkgName bound by spec (via Implicits
// for an unaliased import, or Defs for an aliased one), rewrites every
// matching *ast.SelectorExpr.X identifier found via typesInfo.Uses, and
// finally updates spec.Name itself. This mutates the existing ImportSpec
// node directly instead of going through
// astutil.AddNamedImport/DeleteNamedImport, since a plain rename doesn't
// need to add or remove an import declaration.
func renameImportSpec(typesInfo *types.Info, file *ast.File, spec *ast.ImportSpec, newAlias string) {
	pkgName, ok := typesInfo.Implicits[spec].(*types.PkgName)
	if !ok && spec.Name != nil {
		pkgName, ok = typesInfo.Defs[spec.Name].(*types.PkgName)
	}

	if ok {
		ast.Inspect(file, func(n ast.Node) bool {
			sel, isSel := n.(*ast.SelectorExpr)
			if !isSel {
				return true
			}
			ident, isIdent := sel.X.(*ast.Ident)
			if !isIdent {
				return true
			}
			if use, ok := typesInfo.Uses[ident]; ok && use == pkgName {
				ident.Name = newAlias
			}
			return true
		})
	}

	if newAlias == "" {
		spec.Name = nil
	} else {
		spec.Name = ast.NewIdent(newAlias)
	}
}
