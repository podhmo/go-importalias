package importalias_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	importalias "github.com/podhmo/go-importalias"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), importalias.Analyzer, "dup")
}

// TestAnalyzer_AliasUsedForMultiplePaths covers FR-6.11: the same alias name
// mapped to two distinct import paths within a package.
func TestAnalyzer_AliasUsedForMultiplePaths(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), importalias.Analyzer, "samealias")
}

// TestAnalyzer_DuplicateImportInSameFile covers FR-6.16: the same import
// path aliased more than once within a single file. This is scoped
// separately from FR-6.10 (dup fixture, above): a change here must not make
// that fixture start reporting unexpected duplicates, and vice versa.
func TestAnalyzer_DuplicateImportInSameFile(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), importalias.Analyzer, "dupinfile")
}

// TestAnalyzer_SkipsGeneratedByDefault covers FR-5.7 / DEC-2.4: with the
// default skip_generated=true, the generated file's minority mapping is not
// counted, so the package is consistent and nothing is reported.
func TestAnalyzer_SkipsGeneratedByDefault(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), importalias.Analyzer, "gendefault")
}

// TestAnalyzer_IncludesGeneratedWhenDisabled covers the -skip_generated=false
// path: the generated file is analyzed and its minority mapping is reported.
func TestAnalyzer_IncludesGeneratedWhenDisabled(t *testing.T) {
	if err := importalias.Analyzer.Flags.Set("skip_generated", "false"); err != nil {
		t.Fatalf("set skip_generated=false: %v", err)
	}
	t.Cleanup(func() {
		if err := importalias.Analyzer.Flags.Set("skip_generated", "true"); err != nil {
			t.Fatalf("reset skip_generated=true: %v", err)
		}
	})
	analysistest.Run(t, analysistest.TestData(), importalias.Analyzer, "genreported")
}

func TestAnalyzer_ConfigAliasOverridesMajority(t *testing.T) {
	moduleDir := t.TempDir()
	writeConfig(t, moduleDir, `{"packages":{"example.com/p":{"fmt":""}}}`)

	diags := runAnalyzer(t, "example.com/p", moduleDir, inconsistentSources())
	if got, want := len(diags), 2; got != want {
		t.Fatalf("diagnostics = %d, want %d: %+v", got, want, diags)
	}
	for _, diag := range diags {
		if !strings.Contains(diag.Message, `should use alias no alias, not "f"`) {
			t.Fatalf("diagnostic message = %q, want config-selected no-alias target", diag.Message)
		}
	}
}

func TestAnalyzer_ConfigIgnoreSkipsPackage(t *testing.T) {
	moduleDir := t.TempDir()
	writeConfig(t, moduleDir, `{"ignore":["example.com/..."],"packages":{}}`)

	diags := runAnalyzer(t, "example.com/p", moduleDir, inconsistentSources())
	if len(diags) != 0 {
		t.Fatalf("diagnostics = %+v, want none for ignored package", diags)
	}
}

func TestAnalyzer_NoModuleFallsBackToMajority(t *testing.T) {
	diags := runAnalyzer(t, "example.com/p", "", inconsistentSources())
	if got, want := len(diags), 1; got != want {
		t.Fatalf("diagnostics = %d, want %d: %+v", got, want, diags)
	}
	if !strings.Contains(diags[0].Message, `should use alias "f", not no alias`) {
		t.Fatalf("diagnostic message = %q, want majority-selected alias target", diags[0].Message)
	}
}

func TestAnalyzer_StrictTreatsMultiplicityAsTie(t *testing.T) {
	if err := importalias.Analyzer.Flags.Set("strict", "true"); err != nil {
		t.Fatalf("set strict=true: %v", err)
	}
	t.Cleanup(func() {
		if err := importalias.Analyzer.Flags.Set("strict", "false"); err != nil {
			t.Fatalf("reset strict=false: %v", err)
		}
	})

	diags := runAnalyzer(t, "example.com/p", t.TempDir(), inconsistentSources())
	if len(diags) != 0 {
		t.Fatalf("diagnostics = %+v, want none for strict unresolved tie", diags)
	}
}

func inconsistentSources() map[string]string {
	return map[string]string{
		"a.go": `package p

import f "fmt"

func A() {}
`,
		"b.go": `package p

import f "fmt"

func B() {}
`,
		"c.go": `package p

import "fmt"

func C() {}
`,
	}
}

func writeConfig(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "importalias.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write importalias.json: %v", err)
	}
}

func runAnalyzer(t *testing.T, pkgPath, moduleDir string, sources map[string]string) []analysis.Diagnostic {
	t.Helper()

	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(sources))
	for name, src := range sources {
		file, err := parser.ParseFile(fset, filepath.Join(moduleDir, name), src, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		files = append(files, file)
	}

	var module *analysis.Module
	if moduleDir != "" {
		module = &analysis.Module{Dir: moduleDir}
	}

	var diags []analysis.Diagnostic
	pass := &analysis.Pass{
		Analyzer:  importalias.Analyzer,
		Fset:      fset,
		Files:     files,
		Pkg:       types.NewPackage(pkgPath, "p"),
		TypesInfo: nil,
		Module:    module,
		Report: func(diag analysis.Diagnostic) {
			diags = append(diags, diag)
		},
	}
	if _, err := importalias.Analyzer.Run(pass); err != nil {
		t.Fatalf("run analyzer: %v", err)
	}
	return diags
}
