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

	"github.com/podhmo/go-importalias/internal/decide"
	"github.com/podhmo/go-importalias/internal/fix"
	"github.com/podhmo/go-importalias/internal/scan"
)

func TestApplyToFile_DuplicateImportMerge(t *testing.T) {
	cases := []struct {
		name    string
		dir     string
		pkg     string
		files   []string
		changed map[string]bool
	}{
		{
			name:    "merge_duplicate_to_canonical",
			dir:     "../../testdata/fix/duplicate_import_merge",
			pkg:     "duplicateimportmerge",
			files:   []string{"foo.go", "bar.go"},
			changed: map[string]bool{"foo.go": true},
		},
		{
			name:    "skip_when_canonical_collides",
			dir:     "../../testdata/fix/duplicate_import_collision",
			pkg:     "duplicateimportcollision",
			files:   []string{"main.go", "other.go"},
			changed: map[string]bool{},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fset := token.NewFileSet()
			files := make([]*ast.File, len(c.files))
			for i, name := range c.files {
				f, err := parser.ParseFile(fset, filepath.Join(c.dir, "input", name), nil, parser.ParseComments)
				if err != nil {
					t.Fatalf("ParseFile(%s): %v", name, err)
				}
				files[i] = f
			}

			info := &types.Info{
				Defs:      map[*ast.Ident]types.Object{},
				Uses:      map[*ast.Ident]types.Object{},
				Implicits: map[ast.Node]types.Object{},
				Scopes:    map[ast.Node]*types.Scope{},
			}
			conf := types.Config{Importer: importer.Default()}
			if _, err := conf.Check(c.pkg, fset, files, info); err != nil {
				t.Fatalf("types.Config.Check input: %v", err)
			}

			occs := scan.FromFiles(fset, files, info, scan.Options{Package: c.pkg})
			decisions, collisions, duplicates := decide.Decide(occs, nil, decide.Options{})
			if len(collisions) != 0 {
				t.Fatalf("Decide reported %d alias collisions, want 0: %+v", len(collisions), collisions)
			}
			if len(duplicates) != 1 {
				t.Fatalf("Decide returned %d duplicate imports, want 1: %+v", len(duplicates), duplicates)
			}

			for i, name := range c.files {
				got, changed, err := fix.ApplyToFileWithDuplicates(fset, files[i], info, decisions, duplicates)
				if err != nil {
					t.Fatalf("ApplyToFileWithDuplicates(%s): %v", name, err)
				}
				wantChanged := c.changed[name]
				if changed != wantChanged {
					t.Fatalf("ApplyToFileWithDuplicates(%s) changed = %v, want %v", name, changed, wantChanged)
				}

				want, err := os.ReadFile(filepath.Join(c.dir, "golden", name))
				if err != nil {
					t.Fatalf("read golden %s: %v", name, err)
				}
				if changed && string(got) != string(want) {
					t.Fatalf("ApplyToFileWithDuplicates(%s) output mismatch:\ngot:\n%s\nwant:\n%s", name, got, want)
				}
				if !changed {
					input, err := os.ReadFile(filepath.Join(c.dir, "input", name))
					if err != nil {
						t.Fatalf("read input %s: %v", name, err)
					}
					if string(input) != string(want) {
						t.Fatalf("unchanged fixture %s has input/golden mismatch", name)
					}
				}
			}

			goldenFiles := make([]*ast.File, len(c.files))
			goldenFset := token.NewFileSet()
			for i, name := range c.files {
				f, err := parser.ParseFile(goldenFset, filepath.Join(c.dir, "golden", name), nil, parser.ParseComments)
				if err != nil {
					t.Fatalf("ParseFile golden %s: %v", name, err)
				}
				goldenFiles[i] = f
			}
			if _, err := conf.Check(c.pkg, goldenFset, goldenFiles, nil); err != nil {
				t.Fatalf("types.Config.Check golden: %v", err)
			}
		})
	}
}
