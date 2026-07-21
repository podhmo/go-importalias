package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
	gotHashes := hashGoFiles(t, moduleDir)
	if len(gotHashes) != len(wantHashes) {
		t.Fatalf("go file count = %d, want %d", len(gotHashes), len(wantHashes))
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
