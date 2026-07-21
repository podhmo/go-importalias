package fix_test

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/podhmo/go-importalias/internal/fix"
	"github.com/podhmo/go-importalias/internal/shape"
)

// TestApplyToFile_RenameToMajority type-checks a single-file fixture with
// go/types (no go/packages — that loading is the CLI's responsibility, per
// DEC-11.3) and checks ApplyToFile's output against a golden file.
func TestApplyToFile_RenameToMajority(t *testing.T) {
	const dir = "../../testdata/fix/rename_to_majority"

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join(dir, "input", "main.go"), nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	info := &types.Info{
		Defs:      map[*ast.Ident]types.Object{},
		Uses:      map[*ast.Ident]types.Object{},
		Implicits: map[ast.Node]types.Object{},
	}
	conf := types.Config{Importer: importer.Default()}
	if _, err := conf.Check("main", fset, []*ast.File{file}, info); err != nil {
		t.Fatalf("types.Config.Check: %v", err)
	}

	if len(file.Imports) != 1 {
		t.Fatalf("fixture must have exactly one import, got %d", len(file.Imports))
	}
	decisions := []shape.Decision{
		{
			Package:   "main",
			Path:      "fmt",
			WantAlias: "f",
			Inconsistent: []shape.Occurrence{
				{Package: "main", Path: "fmt", Alias: "", Pos: file.Imports[0].Pos()},
			},
		},
	}

	got, changed, err := fix.ApplyToFile(fset, file, info, decisions)
	if err != nil {
		t.Fatalf("ApplyToFile: %v", err)
	}
	if !changed {
		t.Fatalf("ApplyToFile reported no change")
	}

	want, err := os.ReadFile(filepath.Join(dir, "golden", "main.go"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("ApplyToFile output mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
