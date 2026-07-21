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

// TestPipeline_MultiFileMajority exercises scan -> decide -> fix across
// multiple files of one package, the way the go vet Analyzer and the CLI
// both would: majority vote can only be determined by looking at all of a
// package's files together, and only the file(s) whose alias lost the vote
// should be rewritten (foo.go/bar.go use "f" twice, boo.go uses "oldf" once
// -> "f" wins, only boo.go changes). See docs/02notice.md round 7.
func TestPipeline_MultiFileMajority(t *testing.T) {
	const (
		dir = "../../testdata/fix/multi_file_majority"
		pkg = "multifilemajority"
	)
	names := []string{"foo.go", "bar.go", "boo.go"}

	fset := token.NewFileSet()
	files := make([]*ast.File, len(names))
	for i, name := range names {
		f, err := parser.ParseFile(fset, filepath.Join(dir, "input", name), nil, parser.ParseComments)
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
	if _, err := conf.Check(pkg, fset, files, info); err != nil {
		t.Fatalf("types.Config.Check: %v", err)
	}

	occs := scan.FromFiles(fset, files, info, scan.Options{Package: pkg})
	decisions, collisions, _ := decide.Decide(occs, nil, decide.Options{})
	if len(collisions) != 0 {
		t.Fatalf("Decide reported %d alias collisions, want 0 (this fixture only exercises FR-6.10 majority voting): %+v", len(collisions), collisions)
	}

	if len(decisions) != 1 {
		t.Fatalf("Decide returned %d decisions, want 1 (one import path: fmt)", len(decisions))
	}
	d := decisions[0]
	if d.Tie {
		t.Fatalf("Decide reported a tie, want a clear majority: %+v", d)
	}
	if d.WantAlias != "f" {
		t.Fatalf("WantAlias = %q, want %q (2 votes for \"f\" vs 1 for \"oldf\")", d.WantAlias, "f")
	}
	if len(d.Inconsistent) != 1 || d.Inconsistent[0].Alias != "oldf" {
		t.Fatalf("Inconsistent = %+v, want exactly the boo.go \"oldf\" occurrence", d.Inconsistent)
	}
	if got := len(d.Inconsistent[0].UsePos); got != 2 {
		t.Fatalf("Inconsistent occurrence UsePos has %d entries, want 2 (boo.go uses oldf twice)", got)
	}

	for i, name := range names {
		got, changed, err := fix.ApplyToFile(fset, files[i], info, decisions)
		if err != nil {
			t.Fatalf("ApplyToFile(%s): %v", name, err)
		}

		want, err := os.ReadFile(filepath.Join(dir, "golden", name))
		if err != nil {
			t.Fatalf("read golden %s: %v", name, err)
		}

		wantChanged := name == "boo.go"
		if changed != wantChanged {
			t.Fatalf("ApplyToFile(%s) changed = %v, want %v", name, changed, wantChanged)
		}
		if !changed {
			continue // unchanged files produce no output; nothing to diff
		}
		if string(got) != string(want) {
			t.Fatalf("ApplyToFile(%s) output mismatch:\ngot:\n%s\nwant:\n%s", name, got, want)
		}
	}
}
