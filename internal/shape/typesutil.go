package shape

import (
	"go/ast"
	"go/token"
	"go/types"
)

// PkgNameOf resolves the *types.PkgName that spec binds, given type-check
// info for the file spec belongs to. It returns nil if info is nil or info
// doesn't describe spec (e.g. spec is from a file/package info wasn't
// computed for).
func PkgNameOf(info *types.Info, spec *ast.ImportSpec) *types.PkgName {
	if info == nil {
		return nil
	}
	if pn, ok := info.Implicits[spec].(*types.PkgName); ok {
		return pn
	}
	if spec.Name != nil {
		if pn, ok := info.Defs[spec.Name].(*types.PkgName); ok {
			return pn
		}
	}
	return nil
}

// SelectorIdentsOf returns, in source order, every *ast.SelectorExpr.X
// identifier in file that resolves (via info.Uses) to pkgName — i.e. every
// qualified-identifier reference to that import.
func SelectorIdentsOf(file *ast.File, info *types.Info, pkgName *types.PkgName) []*ast.Ident {
	if info == nil || pkgName == nil {
		return nil
	}
	var idents []*ast.Ident
	ast.Inspect(file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if use, ok := info.Uses[ident]; ok && use == pkgName {
			idents = append(idents, ident)
		}
		return true
	})
	return idents
}

// NameVisibleAt reports whether name is already bound, at pos, to an object
// other than except — i.e. whether introducing (or renaming something to)
// an identifier spelled name at pos would collide with a local variable,
// another import, a top-level declaration, etc. already visible there. It
// walks from the innermost scope containing pos up through (and including)
// the package scope; the universe scope (predeclared identifiers such as
// "len") is not checked.
func NameVisibleAt(info *types.Info, file *ast.File, pos token.Pos, name string, except types.Object) bool {
	if info == nil {
		return false
	}
	fileScope := info.Scopes[file]
	if fileScope == nil {
		return false
	}
	scope := fileScope.Innermost(pos)
	if scope == nil {
		scope = fileScope
	}
	for s := scope; s != nil; s = s.Parent() {
		if s.Parent() == nil {
			break // universe scope: predeclared identifiers are not checked
		}
		if obj := s.Lookup(name); obj != nil && obj != except {
			return true
		}
	}
	return false
}
