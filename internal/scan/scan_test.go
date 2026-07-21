package scan_test

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/podhmo/go-importalias/internal/scan"
)

// TestFromFiles_ExcludesBlankAndDotImports covers DEC-11.19: blank ("_") and
// dot (".") imports must not become Occurrences, since letting them leak
// into internal/decide's majority vote would corrupt it.
func TestFromFiles_ExcludesBlankAndDotImports(t *testing.T) {
	const src = `package p

import (
	f "fmt"
	_ "os"
	. "strings"
)

func F() {
	f.Println(Repeat("x", 1))
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	info := &types.Info{
		Defs:      map[*ast.Ident]types.Object{},
		Uses:      map[*ast.Ident]types.Object{},
		Implicits: map[ast.Node]types.Object{},
		Scopes:    map[ast.Node]*types.Scope{},
	}
	conf := types.Config{Importer: importer.Default()}
	if _, err := conf.Check("p", fset, []*ast.File{file}, info); err != nil {
		t.Fatalf("types.Config.Check: %v", err)
	}

	occs := scan.FromFiles(fset, []*ast.File{file}, info, scan.Options{Package: "p"})

	if len(occs) != 1 {
		t.Fatalf("FromFiles returned %d occurrences, want 1 (only the \"fmt\" import): %+v", len(occs), occs)
	}
	if got, want := occs[0].Path, "fmt"; got != want {
		t.Fatalf("occs[0].Path = %q, want %q", got, want)
	}
	if got, want := occs[0].Alias, "f"; got != want {
		t.Fatalf("occs[0].Alias = %q, want %q", got, want)
	}
	if got := len(occs[0].UsePos); got != 1 {
		t.Fatalf("occs[0].UsePos has %d entries, want 1 (fmt.Println is used once)", got)
	}
}
