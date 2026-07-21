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

// TestApplyToFile_Collision covers FR-7.21's collision check for the plain
// rename case (not just the alias-removal case it was originally scoped
// to): renaming an import's qualifier must never be applied if the
// resulting name is already bound to something else (a local variable, in
// these fixtures) at any of its use sites — in both directions (unaliased
// "fmt" -> alias "f", and alias "f" -> unaliased "fmt"). A pre-existing,
// unrelated same-named local variable elsewhere in the file (a different,
// non-overlapping scope) must NOT block the rewrite — that's just ordinary
// Go shadowing and isn't this tool's concern. It also covers the successful
// (no collision) alias-removal direction end to end, which earlier rounds
// only ever exercised via the collision-blocked case — see docs/02notice.md
// round 9.
func TestApplyToFile_Collision(t *testing.T) {
	cases := []struct {
		name        string
		dir         string
		wantAlias   string // "" means the fix is supposed to unalias the import
		wantChanged bool
	}{
		{
			name:        "unaliased_to_alias_collides_with_local_var",
			dir:         "../../testdata/fix/collision_unaliased_to_alias",
			wantAlias:   "f",
			wantChanged: false,
		},
		{
			name:        "alias_to_unaliased_collides_with_local_var",
			dir:         "../../testdata/fix/collision_alias_to_unaliased",
			wantAlias:   "",
			wantChanged: false,
		},
		{
			name:        "unrelated_same_named_var_in_other_scope_does_not_block",
			dir:         "../../testdata/fix/no_collision_unrelated_scope",
			wantAlias:   "f",
			wantChanged: true,
		},
		{
			name:        "alias_to_unaliased_no_collision",
			dir:         "../../testdata/fix/unalias_no_collision",
			wantAlias:   "",
			wantChanged: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, filepath.Join(c.dir, "input", "main.go"), nil, parser.ParseComments)
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
					WantAlias: c.wantAlias,
					Inconsistent: []shape.Occurrence{
						{Package: "main", Path: "fmt", Pos: file.Imports[0].Pos()},
					},
				},
			}

			got, changed, err := fix.ApplyToFile(fset, file, info, decisions)
			if err != nil {
				t.Fatalf("ApplyToFile: %v", err)
			}
			if changed != c.wantChanged {
				t.Fatalf("ApplyToFile changed = %v, want %v", changed, c.wantChanged)
			}

			want, err := os.ReadFile(filepath.Join(c.dir, "golden", "main.go"))
			if err != nil {
				t.Fatalf("read golden: %v", err)
			}
			if !changed {
				return // unchanged: nothing to diff, golden is just a copy of input
			}
			if string(got) != string(want) {
				t.Fatalf("ApplyToFile output mismatch:\ngot:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}
