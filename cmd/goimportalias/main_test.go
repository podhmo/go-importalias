package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestLooksLikeVetToolInvocation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "version handshake", args: []string{"goimportalias", "-V=full"}, want: true},
		{name: "flags handshake", args: []string{"goimportalias", "-flags"}, want: true},
		{name: "cfg", args: []string{"goimportalias", filepath.Join(t.TempDir(), "vet.cfg")}, want: true},
		{name: "json cfg", args: []string{"goimportalias", "-json", filepath.Join(t.TempDir(), "vet.cfg")}, want: true},
		{name: "analyzer flag cfg", args: []string{"goimportalias", "-importalias.include_tests=false", filepath.Join(t.TempDir(), "vet.cfg")}, want: true},
		{name: "json analyzer flag cfg", args: []string{"goimportalias", "-json", "-importalias.strict", filepath.Join(t.TempDir(), "vet.cfg")}, want: true},
		{name: "no args", args: []string{"goimportalias"}, want: false},
		{name: "cli arg", args: []string{"goimportalias", "./..."}, want: false},
		{name: "multiple args", args: []string{"goimportalias", "-V=full", "extra"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikeVetToolInvocation(tt.args); got != tt.want {
				t.Fatalf("looksLikeVetToolInvocation(%q) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestVetToolReportsInconsistentImports(t *testing.T) {
	tool := buildVetTool(t)
	moduleDir := writeVetModule(t, map[string]string{
		"a.go": `package p

import f "fmt"

func A() { f.Println("a") }
`,
		"b.go": `package p

import "fmt"

func B() { fmt.Println("b") }
`,
		"c.go": `package p

import f "fmt"

func C() { f.Println("c") }
`,
	})
	sourceHashes := hashGoFiles(t, moduleDir)

	cmd := exec.Command("go", "vet", "-vettool="+tool, "./...")
	cmd.Dir = moduleDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatalf("go vet succeeded, want non-zero exit; stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if _, ok := err.(*exec.ExitError); !ok {
		t.Fatalf("go vet failed unexpectedly: %v", err)
	}
	if !strings.Contains(stderr.String(), `should use alias "f", not no alias`) {
		t.Fatalf("stderr = %q, want importalias diagnostic", stderr.String())
	}
	assertNoVetWrites(t, moduleDir, sourceHashes)
}

func TestVetToolSucceedsForConsistentImports(t *testing.T) {
	tool := buildVetTool(t)
	moduleDir := writeVetModule(t, map[string]string{
		"a.go": `package p

import f "fmt"

func A() { f.Println("a") }
`,
		"b.go": `package p

import f "fmt"

func B() { f.Println("b") }
`,
	})
	sourceHashes := hashGoFiles(t, moduleDir)

	cmd := exec.Command("go", "vet", "-vettool="+tool, "./...")
	cmd.Dir = moduleDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go vet failed: %v\n%s", err, out)
	}
	assertNoVetWrites(t, moduleDir, sourceHashes)
}

func TestCLIScanReportsInconsistentImports(t *testing.T) {
	tool := buildVetTool(t)
	moduleDir := writeVetModule(t, map[string]string{
		"a.go": `package p

import f "fmt"

func A() { f.Println("a") }
`,
		"b.go": `package p

import "fmt"

func B() { fmt.Println("b") }
`,
		"c.go": `package p

import f "fmt"

func C() { f.Println("c") }
`,
	})
	sourceHashes := hashGoFiles(t, moduleDir)

	cmd := exec.Command(tool, "./...")
	cmd.Dir = moduleDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if got := exitCode(err); got != 1 {
		t.Fatalf("goimportalias exit = %d, want 1; stdout=%q stderr=%q err=%v", got, stdout.String(), stderr.String(), err)
	}
	if !strings.Contains(stdout.String(), `should use alias "f", not no alias`) {
		t.Fatalf("stdout = %q, want importalias diagnostic", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	assertGoFileHashes(t, moduleDir, sourceHashes)
	assertConfigContent(t, filepath.Join(moduleDir, "importalias.json"), `{
  "packages": {
    "example.com/vetfixture": {
      "fmt": "f"
    }
  }
}
`)
}

func TestCLIScanReportsInconsistentImportsWhenNoAliasWins(t *testing.T) {
	tool := buildVetTool(t)
	moduleDir := writeVetModule(t, map[string]string{
		"a.go": `package p

import "fmt"

func A() { fmt.Println("a") }
`,
		"b.go": `package p

import f "fmt"

func B() { f.Println("b") }
`,
		"c.go": `package p

import "fmt"

func C() { fmt.Println("c") }
`,
	})
	sourceHashes := hashGoFiles(t, moduleDir)

	cmd := exec.Command(tool, "./...")
	cmd.Dir = moduleDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if got := exitCode(err); got != 1 {
		t.Fatalf("goimportalias exit = %d, want 1; stdout=%q stderr=%q err=%v", got, stdout.String(), stderr.String(), err)
	}
	if !strings.Contains(stdout.String(), `should use alias no alias, not "f"`) {
		t.Fatalf("stdout = %q, want no-alias diagnostic", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	assertGoFileHashes(t, moduleDir, sourceHashes)
	assertConfigContent(t, filepath.Join(moduleDir, "importalias.json"), `{
  "packages": {}
}
`)
}

func TestCLIScanReportsNoAliasMajorityWithOmittedPackageConfig(t *testing.T) {
	tool := buildVetTool(t)
	moduleDir := writeVetModule(t, map[string]string{
		"a.go": `package p

import "fmt"

func A() { fmt.Println("a") }
`,
		"b.go": `package p

import f "fmt"

func B() { f.Println("b") }
`,
		"c.go": `package p

import "fmt"

func C() { fmt.Println("c") }
`,
	})
	sourceHashes := hashGoFiles(t, moduleDir)
	configPath := filepath.Join(moduleDir, "importalias.json")
	if err := os.WriteFile(configPath, []byte(`{
  "packages": {}
}
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := exec.Command(tool, "./...")
	cmd.Dir = moduleDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if got := exitCode(err); got != 1 {
		t.Fatalf("goimportalias exit = %d, want 1; stdout=%q stderr=%q err=%v", got, stdout.String(), stderr.String(), err)
	}
	if !strings.Contains(stdout.String(), `should use alias no alias, not "f"`) {
		t.Fatalf("stdout = %q, want no-alias diagnostic", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	assertGoFileHashes(t, moduleDir, sourceHashes)
	assertConfigContent(t, configPath, `{
  "packages": {}
}
`)
}

func TestCLIScanReportsTie(t *testing.T) {
	tool := buildVetTool(t)
	moduleDir := writeVetModule(t, map[string]string{
		"a.go": `package p

import f "fmt"

func A() { f.Println("a") }
`,
		"b.go": `package p

import g "fmt"

func B() { g.Println("b") }
`,
	})
	sourceHashes := hashGoFiles(t, moduleDir)

	cmd := exec.Command(tool, "./...")
	cmd.Dir = moduleDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if got := exitCode(err); got != 1 {
		t.Fatalf("goimportalias exit = %d, want 1; stdout=%q stderr=%q err=%v", got, stdout.String(), stderr.String(), err)
	}
	if !strings.Contains(stdout.String(), `import "fmt" has unresolved tie among aliases ["f", "g"]`) {
		t.Fatalf("stdout = %q, want tie diagnostic", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	assertGoFileHashes(t, moduleDir, sourceHashes)
	assertConfigContent(t, filepath.Join(moduleDir, "importalias.json"), `{
  "packages": {
    "example.com/vetfixture": {
      "fmt": [
        "f",
        "g"
      ]
    }
  }
}
`)
}

func TestCLIScanSucceedsForConsistentImports(t *testing.T) {
	tool := buildVetTool(t)
	moduleDir := writeVetModule(t, map[string]string{
		"a.go": `package p

import f "fmt"

func A() { f.Println("a") }
`,
		"b.go": `package p

import f "fmt"

func B() { f.Println("b") }
`,
	})
	sourceHashes := hashGoFiles(t, moduleDir)

	cmd := exec.Command(tool, "./...")
	cmd.Dir = moduleDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if got := exitCode(err); got != 0 {
		t.Fatalf("goimportalias exit = %d, want 0; stdout=%q stderr=%q err=%v", got, stdout.String(), stderr.String(), err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q, want both empty", stdout.String(), stderr.String())
	}
	assertGoFileHashes(t, moduleDir, sourceHashes)
	assertConfigContent(t, filepath.Join(moduleDir, "importalias.json"), `{
  "packages": {
    "example.com/vetfixture": {
      "fmt": "f"
    }
  }
}
`)
}

func TestCLIScanLoadFailureExitCode(t *testing.T) {
	tool := buildVetTool(t)
	moduleDir := writeVetModule(t, map[string]string{
		"a.go": `package p
`,
	})

	cmd := exec.Command(tool, "./does-not-exist")
	cmd.Dir = moduleDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if got := exitCode(err); got != 2 {
		t.Fatalf("goimportalias exit = %d, want 2; stdout=%q stderr=%q err=%v", got, stdout.String(), stderr.String(), err)
	}
	if stderr.Len() == 0 {
		t.Fatalf("stderr is empty, want load failure")
	}
}

func TestCLIScanFindsModuleRootFromSubdirectory(t *testing.T) {
	tool := buildVetTool(t)
	moduleDir := writeVetModule(t, map[string]string{
		"sub/a.go": `package sub

import f "fmt"

func A() { f.Println("a") }
`,
		"sub/b.go": `package sub

import "fmt"

func B() { fmt.Println("b") }
`,
		"sub/c.go": `package sub

import f "fmt"

func C() { f.Println("c") }
`,
	})
	sourceHashes := hashGoFiles(t, filepath.Join(moduleDir, "sub"))

	cmd := exec.Command(tool, ".")
	cmd.Dir = filepath.Join(moduleDir, "sub")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if got := exitCode(err); got != 1 {
		t.Fatalf("goimportalias exit = %d, want 1; stdout=%q stderr=%q err=%v", got, stdout.String(), stderr.String(), err)
	}
	if !strings.Contains(stdout.String(), `should use alias "f", not no alias`) {
		t.Fatalf("stdout = %q, want importalias diagnostic", stdout.String())
	}
	assertGoFileHashes(t, filepath.Join(moduleDir, "sub"), sourceHashes)
	assertConfigContent(t, filepath.Join(moduleDir, "importalias.json"), `{
  "packages": {
    "example.com/vetfixture/sub": {
      "fmt": "f"
    }
  }
}
`)
}

func TestCLIConfigMergesExistingFile(t *testing.T) {
	tool := buildVetTool(t)
	moduleDir := writeVetModule(t, map[string]string{
		"a.go": `package p

import f "fmt"

func A() { f.Println("a") }
`,
		"b.go": `package p

import f "fmt"

func B() { f.Println("b") }
`,
	})
	sourceHashes := hashGoFiles(t, moduleDir)
	configPath := filepath.Join(moduleDir, "importalias.json")
	if err := os.WriteFile(configPath, []byte(`{
  "packages": {
    "example.com/other": {
      "os": "o"
    },
    "example.com/vetfixture": {
      "old/path": "old"
    }
  },
  "ignore": [
    "example.com/legacy"
  ]
}
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := exec.Command(tool, "./...")
	cmd.Dir = moduleDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if got := exitCode(err); got != 0 {
		t.Fatalf("goimportalias exit = %d, want 0; stdout=%q stderr=%q err=%v", got, stdout.String(), stderr.String(), err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q, want both empty", stdout.String(), stderr.String())
	}
	assertGoFileHashes(t, moduleDir, sourceHashes)
	assertConfigContent(t, configPath, `{
  "packages": {
    "example.com/other": {
      "os": "o"
    },
    "example.com/vetfixture": {
      "fmt": "f"
    }
  },
  "ignore": [
    "example.com/legacy"
  ]
}
`)
}

func TestCLIConfigFlagWritesExplicitPath(t *testing.T) {
	tool := buildVetTool(t)
	moduleDir := writeVetModule(t, map[string]string{
		"a.go": `package p

import "fmt"

func A() { fmt.Println("a") }
`,
	})
	sourceHashes := hashGoFiles(t, moduleDir)
	configPath := filepath.Join(t.TempDir(), "custom-importalias.json")

	cmd := exec.Command(tool, "-config", configPath, "./...")
	cmd.Dir = moduleDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if got := exitCode(err); got != 0 {
		t.Fatalf("goimportalias exit = %d, want 0; stdout=%q stderr=%q err=%v", got, stdout.String(), stderr.String(), err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q, want both empty", stdout.String(), stderr.String())
	}
	assertGoFileHashes(t, moduleDir, sourceHashes)
	if _, err := os.Stat(filepath.Join(moduleDir, "importalias.json")); !os.IsNotExist(err) {
		t.Fatalf("default importalias.json stat error = %v, want not exist", err)
	}
	assertConfigContent(t, configPath, `{
  "packages": {}
}
`)
}

func TestCLIFixAppliesRenameAndExitsZero(t *testing.T) {
	tool := buildVetTool(t)
	moduleDir := writeVetModule(t, map[string]string{
		"a.go": `package p

import f "fmt"

func A() { f.Println("a") }
`,
		"b.go": `package p

import "fmt"

func B() { fmt.Println("b") }
`,
		"c.go": `package p

import f "fmt"

func C() { f.Println("c") }
`,
	})
	before := hashGoFiles(t, moduleDir)

	cmd := exec.Command(tool, "-fix", "./...")
	cmd.Dir = moduleDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if got := exitCode(err); got != 0 {
		t.Fatalf("goimportalias -fix exit = %d, want 0; stdout=%q stderr=%q err=%v", got, stdout.String(), stderr.String(), err)
	}
	if !strings.Contains(stdout.String(), "b.go: applied import alias fix") {
		t.Fatalf("stdout = %q, want applied fix message for b.go", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	assertFileContent(t, filepath.Join(moduleDir, "b.go"), `package p

import f "fmt"

func B() { f.Println("b") }
`)
	after := hashGoFiles(t, moduleDir)
	for _, name := range []string{"a.go", "c.go"} {
		if after[name] != before[name] {
			t.Fatalf("%s changed after goimportalias -fix, want unchanged", name)
		}
	}
	if after["b.go"] == before["b.go"] {
		t.Fatalf("b.go did not change after goimportalias -fix")
	}
	assertConfigContent(t, filepath.Join(moduleDir, "importalias.json"), `{
  "packages": {
    "example.com/vetfixture": {
      "fmt": "f"
    }
  }
}
`)
}

func TestCLIFixLeavesTieUnchangedAndExitsOne(t *testing.T) {
	tool := buildVetTool(t)
	moduleDir := writeVetModule(t, map[string]string{
		"a.go": `package p

import f "fmt"

func A() { f.Println("a") }
`,
		"b.go": `package p

import g "fmt"

func B() { g.Println("b") }
`,
	})
	sourceHashes := hashGoFiles(t, moduleDir)

	cmd := exec.Command(tool, "-fix", "./...")
	cmd.Dir = moduleDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if got := exitCode(err); got != 1 {
		t.Fatalf("goimportalias -fix exit = %d, want 1; stdout=%q stderr=%q err=%v", got, stdout.String(), stderr.String(), err)
	}
	if !strings.Contains(stdout.String(), `import "fmt" has unresolved tie among aliases ["f", "g"]`) {
		t.Fatalf("stdout = %q, want tie diagnostic", stdout.String())
	}
	if strings.Contains(stdout.String(), "applied import alias fix") {
		t.Fatalf("stdout = %q, want no applied fix message", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	assertGoFileHashes(t, moduleDir, sourceHashes)
	assertConfigContent(t, filepath.Join(moduleDir, "importalias.json"), `{
  "packages": {
    "example.com/vetfixture": {
      "fmt": [
        "f",
        "g"
      ]
    }
  }
}
`)
}

func TestCLIFixLeavesCollisionUnchangedAndExitsOne(t *testing.T) {
	tool := buildVetTool(t)
	moduleDir := writeVetModule(t, map[string]string{
		"a.go": `package p

import f "fmt"

func A() { f.Println("a") }
`,
		"b.go": `package p

import g "fmt"

func B() {
	f := "shadow"
	g.Println(f)
}
`,
		"c.go": `package p

import f "fmt"

func C() { f.Println("c") }
`,
	})
	sourceHashes := hashGoFiles(t, moduleDir)

	cmd := exec.Command(tool, "-fix", "./...")
	cmd.Dir = moduleDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if got := exitCode(err); got != 1 {
		t.Fatalf("goimportalias -fix exit = %d, want 1; stdout=%q stderr=%q err=%v", got, stdout.String(), stderr.String(), err)
	}
	if !strings.Contains(stdout.String(), `should use alias "f", not "g"`) {
		t.Fatalf("stdout = %q, want remaining diagnostic", stdout.String())
	}
	if strings.Contains(stdout.String(), "applied import alias fix") {
		t.Fatalf("stdout = %q, want no applied fix message", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	assertGoFileHashes(t, moduleDir, sourceHashes)
	assertConfigContent(t, filepath.Join(moduleDir, "importalias.json"), `{
  "packages": {
    "example.com/vetfixture": {
      "fmt": "f"
    }
  }
}
`)
}

// TestCLIScanExcludesTestFilesByDefault covers the go/packages default:
// without -include-tests the CLI never loads *_test.go files, so an
// inconsistency that only exists in a test file is not reported (FR-7.9's
// target set is opt-in).
func TestCLIScanExcludesTestFilesByDefault(t *testing.T) {
	tool := buildVetTool(t)
	moduleDir := writeVetModule(t, map[string]string{
		"a.go": `package p

import f "fmt"

func A() { f.Println("a") }
`,
		"b.go": `package p

import f "fmt"

func B() { f.Println("b") }
`,
		"p_test.go": `package p

import "fmt"
import "testing"

func TestC(t *testing.T) { fmt.Println("c") }
`,
	})
	sourceHashes := hashGoFiles(t, moduleDir)

	cmd := exec.Command(tool, "./...")
	cmd.Dir = moduleDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if got := exitCode(err); got != 0 {
		t.Fatalf("goimportalias exit = %d, want 0; stdout=%q stderr=%q err=%v", got, stdout.String(), stderr.String(), err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q, want both empty", stdout.String(), stderr.String())
	}
	assertGoFileHashes(t, moduleDir, sourceHashes)
	assertConfigContent(t, filepath.Join(moduleDir, "importalias.json"), `{
  "packages": {
    "example.com/vetfixture": {
      "fmt": "f"
    }
  }
}
`)
}

// TestCLIScanIncludeTestsReportsTestFileImports covers -include-tests:
// internal test files join the package's majority vote, and external test
// packages ("<pkg>_test") are scanned as packages of their own, while the
// synthetic "<pkg>.test" binary is not.
func TestCLIScanIncludeTestsReportsTestFileImports(t *testing.T) {
	tool := buildVetTool(t)
	moduleDir := writeVetModule(t, map[string]string{
		"a.go": `package p

import f "fmt"

func A() { f.Println("a") }
`,
		"b.go": `package p

import f "fmt"

func B() { f.Println("b") }
`,
		"p_test.go": `package p

import "fmt"
import "testing"

func TestC(t *testing.T) { fmt.Println("c") }
`,
		"ext_test.go": `package p_test

import o "os"
import "testing"

func TestD(t *testing.T) { _ = o.Getenv("x") }
`,
	})
	sourceHashes := hashGoFiles(t, moduleDir)

	cmd := exec.Command(tool, "-include-tests", "./...")
	cmd.Dir = moduleDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if got := exitCode(err); got != 1 {
		t.Fatalf("goimportalias -include-tests exit = %d, want 1; stdout=%q stderr=%q err=%v", got, stdout.String(), stderr.String(), err)
	}
	if !strings.Contains(stdout.String(), "p_test.go") ||
		!strings.Contains(stdout.String(), `should use alias "f", not no alias`) {
		t.Fatalf("stdout = %q, want importalias diagnostic at p_test.go", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	assertGoFileHashes(t, moduleDir, sourceHashes)
	assertConfigContent(t, filepath.Join(moduleDir, "importalias.json"), `{
  "packages": {
    "example.com/vetfixture": {
      "fmt": "f"
    },
    "example.com/vetfixture_test": {
      "os": "o"
    }
  }
}
`)
}

// TestVetToolIncludeTestsFlag covers the analyzer flag matching go vet's
// default: test files are analyzed by default, and
// -importalias.include_tests=false excludes them.
func TestVetToolIncludeTestsFlag(t *testing.T) {
	tool := buildVetTool(t)
	moduleDir := writeVetModule(t, map[string]string{
		"a.go": `package p

import f "fmt"

func A() { f.Println("a") }
`,
		"b.go": `package p

import f "fmt"

func B() { f.Println("b") }
`,
		"p_test.go": `package p

import "fmt"
import "testing"

func TestC(t *testing.T) { fmt.Println("c") }
`,
	})
	sourceHashes := hashGoFiles(t, moduleDir)

	t.Run("default includes test files", func(t *testing.T) {
		cmd := exec.Command("go", "vet", "-vettool="+tool, "./...")
		cmd.Dir = moduleDir
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err == nil {
			t.Fatalf("go vet succeeded, want non-zero exit; stdout=%q stderr=%q", stdout.String(), stderr.String())
		}
		if _, ok := err.(*exec.ExitError); !ok {
			t.Fatalf("go vet failed unexpectedly: %v", err)
		}
		if !strings.Contains(stderr.String(), "p_test.go") {
			t.Fatalf("stderr = %q, want importalias diagnostic at p_test.go", stderr.String())
		}
	})

	t.Run("include_tests=false excludes test files", func(t *testing.T) {
		cmd := exec.Command("go", "vet", "-vettool="+tool, "-importalias.include_tests=false", "./...")
		cmd.Dir = moduleDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go vet failed: %v\n%s", err, out)
		}
	})

	assertNoVetWrites(t, moduleDir, sourceHashes)
}

func TestSelectScanPackages(t *testing.T) {
	plain := &packages.Package{PkgPath: "example.com/p", Name: "p"}
	augmented := &packages.Package{PkgPath: "example.com/p", Name: "p", ForTest: "example.com/p"}
	external := &packages.Package{PkgPath: "example.com/p_test", Name: "p_test", ForTest: "example.com/p"}
	testMain := &packages.Package{PkgPath: "example.com/p.test", Name: "main"}

	got := selectScanPackages([]*packages.Package{plain, augmented, external, testMain})
	if len(got) != 2 {
		t.Fatalf("selectScanPackages returned %d packages, want 2: %+v", len(got), got)
	}
	if got[0] != augmented {
		t.Fatalf("got[0].PkgPath = %q ForTest = %q, want the test-augmented variant of example.com/p", got[0].PkgPath, got[0].ForTest)
	}
	if got[1] != external {
		t.Fatalf("got[1].PkgPath = %q, want %q", got[1].PkgPath, external.PkgPath)
	}
}

func buildVetTool(t *testing.T) string {
	t.Helper()
	tool := filepath.Join(t.TempDir(), "goimportalias")
	cmd := exec.Command("go", "build", "-o", tool, ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build goimportalias: %v\n%s", err, out)
	}
	return tool
}

func writeVetModule(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/vetfixture\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for name, content := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

func hashGoFiles(t *testing.T, dir string) map[string][32]byte {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob go files: %v", err)
	}
	hashes := make(map[string][32]byte, len(matches))
	for _, name := range matches {
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		hashes[filepath.Base(name)] = sha256.Sum256(content)
	}
	return hashes
}

func assertNoVetWrites(t *testing.T, moduleDir string, wantHashes map[string][32]byte) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(moduleDir, "importalias.json")); !os.IsNotExist(err) {
		t.Fatalf("importalias.json stat error = %v, want not exist", err)
	}
	assertGoFileHashes(t, moduleDir, wantHashes)
}

func assertGoFileHashes(t *testing.T, moduleDir string, wantHashes map[string][32]byte) {
	t.Helper()
	gotHashes := hashGoFiles(t, moduleDir)
	if len(gotHashes) != len(wantHashes) {
		t.Fatalf("go file count in %s = %d, want %d", moduleDir, len(gotHashes), len(wantHashes))
	}
	for name, want := range wantHashes {
		got, ok := gotHashes[name]
		if !ok {
			t.Fatalf("%s missing after go vet", name)
		}
		if got != want {
			t.Fatalf("%s changed after go vet: got %s, want %s", name, fmtHash(got), fmtHash(want))
		}
	}
}

func fmtHash(sum [32]byte) string {
	return fmt.Sprintf("%x", sum[:])
}

func assertConfigContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config %s: %v", path, err)
	}
	if got := string(got); got != want {
		t.Fatalf("config %s =\n%s\nwant\n%s", path, got, want)
	}
	var decoded any
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("config %s is invalid JSON: %v", path, err)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("%s =\n%s\nwant\n%s", path, got, want)
	}
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return -1
}
