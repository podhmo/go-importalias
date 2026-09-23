package main

import (
	"flag"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis/unitchecker"
	"golang.org/x/tools/go/packages"

	importalias "github.com/podhmo/go-importalias"
	"github.com/podhmo/go-importalias/internal/decide"
	"github.com/podhmo/go-importalias/internal/fix"
	"github.com/podhmo/go-importalias/internal/scan"
	"github.com/podhmo/go-importalias/internal/shape"
)

func main() {
	if looksLikeVetToolInvocation(os.Args) {
		unitchecker.Main(importalias.Analyzer)
		return
	}
	os.Exit(runCLI(os.Args[1:]))
}

func looksLikeVetToolInvocation(args []string) bool {
	if len(args) < 2 {
		return false
	}
	if len(args) == 2 && (args[1] == "-V=full" || args[1] == "-flags") {
		return true
	}
	// The unitchecker protocol puts driver flags (-json) and analyzer flags
	// (e.g. -importalias.include_tests=false) before the *.cfg unit file,
	// so flag-shaped arguments may precede it.
	if !strings.HasSuffix(args[len(args)-1], ".cfg") {
		return false
	}
	for _, arg := range args[1 : len(args)-1] {
		if !strings.HasPrefix(arg, "-") {
			return false
		}
	}
	return true
}

func runCLI(args []string) int {
	var opts cliOptions
	fs := flag.NewFlagSet("goimportalias", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.BoolVar(&opts.fix, "fix", false, "apply safe import alias fixes")
	fs.StringVar(&opts.config, "config", "", "path to importalias.json")
	fs.BoolVar(&opts.strict, "strict", false, "treat any multiple aliases for the same import path as an unresolved tie")
	fs.BoolVar(&opts.skipGenerated, "skip-generated", true, "skip files carrying a generated-code marker")
	fs.BoolVar(&opts.includeTests, "include-tests", false, "include *_test.go files (loads test variants via go/packages Tests)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	patterns := fs.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "goimportalias: get working directory: %v\n", err)
		return 2
	}
	moduleRoot, err := findModuleRoot(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "goimportalias: %v\n", err)
		return 2
	}
	if opts.config == "" {
		opts.config = filepath.Join(moduleRoot, "importalias.json")
	}
	cfg, err := shape.Load(opts.config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "goimportalias: %v\n", err)
		return 2
	}

	fset, pkgs, err := loadPackages(cwd, patterns, opts.includeTests)
	if err != nil {
		fmt.Fprintf(os.Stderr, "goimportalias: load packages: %v\n", err)
		return 2
	}
	if n := printPackageErrors(pkgs); n > 0 {
		return 2
	}

	if opts.fix {
		if err := applyFixes(fset, pkgs, cfg, opts); err != nil {
			fmt.Fprintf(os.Stderr, "goimportalias: %v\n", err)
			return 2
		}
		fset, pkgs, err = loadPackages(cwd, patterns, opts.includeTests)
		if err != nil {
			fmt.Fprintf(os.Stderr, "goimportalias: reload packages after fix: %v\n", err)
			return 2
		}
		if n := printPackageErrors(pkgs); n > 0 {
			return 2
		}
	}

	found := false
	fresh := shape.NewFile()
	for _, pkg := range pkgs {
		if cfg.IgnoresPackage(pkg.PkgPath) {
			continue
		}
		occs := scan.FromFiles(fset, pkg.Syntax, pkg.TypesInfo, scan.Options{
			Package:       pkg.PkgPath,
			SkipGenerated: opts.skipGenerated,
			IncludeTests:  opts.includeTests,
		})
		decisions, collisions, duplicates := decide.Decide(occs, cfg, decide.Options{Strict: opts.strict})
		fresh.Packages[pkg.PkgPath] = decisionsToConfig(decisions)
		if reportDecisions(fset, decisions) {
			found = true
		}
		if reportCollisions(fset, collisions) {
			found = true
		}
		if reportDuplicates(fset, duplicates) {
			found = true
		}
	}
	if err := shape.Save(opts.config, shape.Merge(cfg, fresh)); err != nil {
		fmt.Fprintf(os.Stderr, "goimportalias: %v\n", err)
		return 2
	}
	if found {
		return 1
	}
	return 0
}

func loadPackages(cwd string, patterns []string, includeTests bool) (*token.FileSet, []*packages.Package, error) {
	fset := token.NewFileSet()
	pkgs, err := packages.Load(&packages.Config{
		Dir:   cwd,
		Fset:  fset,
		Tests: includeTests,
		Mode: packages.NeedName |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedSyntax |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedForTest,
	}, patterns...)
	if err != nil {
		return nil, nil, err
	}
	if len(pkgs) == 0 {
		return nil, nil, fmt.Errorf("no packages matched")
	}
	return fset, selectScanPackages(pkgs), nil
}

// selectScanPackages reduces what packages.Load returns when Tests is on to
// the packages worth scanning: it drops the synthetic test binaries
// ("<pkg>.test", the generated package main that only exists in the build
// cache) and, for a PkgPath reported both as the plain package and its
// test-augmented variant, keeps the augmented one — its file list is the
// strict superset (non-test files plus internal *_test.go files).
// External test packages ("<pkg>_test") are kept as packages of their own.
func selectScanPackages(pkgs []*packages.Package) []*packages.Package {
	out := make([]*packages.Package, 0, len(pkgs))
	byPath := make(map[string]int, len(pkgs)) // PkgPath -> index into out
	for _, pkg := range pkgs {
		if pkg.Name == "main" && strings.HasSuffix(pkg.PkgPath, ".test") {
			continue
		}
		if i, ok := byPath[pkg.PkgPath]; ok {
			if pkg.ForTest != "" {
				out[i] = pkg // prefer the test-augmented variant
			}
			continue
		}
		byPath[pkg.PkgPath] = len(out)
		out = append(out, pkg)
	}
	return out
}

func applyFixes(fset *token.FileSet, pkgs []*packages.Package, cfg *shape.File, opts cliOptions) error {
	for _, pkg := range pkgs {
		if cfg.IgnoresPackage(pkg.PkgPath) {
			continue
		}
		occs := scan.FromFiles(fset, pkg.Syntax, pkg.TypesInfo, scan.Options{
			Package:       pkg.PkgPath,
			SkipGenerated: opts.skipGenerated,
			IncludeTests:  opts.includeTests,
		})
		decisions, collisions, duplicates := decide.Decide(occs, cfg, decide.Options{Strict: opts.strict})
		for _, file := range pkg.Syntax {
			src, changed, err := fix.ApplyToFileWithCollisionsAndDuplicates(fset, file, pkg.TypesInfo, decisions, collisions, duplicates)
			if err != nil {
				return err
			}
			if !changed {
				continue
			}
			name := fset.Position(file.Pos()).Filename
			if err := os.WriteFile(name, src, 0o644); err != nil {
				return fmt.Errorf("write fixed file %s: %w", name, err)
			}
			fmt.Fprintf(os.Stdout, "%s: applied import alias fix\n", name)
		}
	}
	return nil
}

type cliOptions struct {
	fix           bool
	config        string
	strict        bool
	skipGenerated bool
	includeTests  bool
}

func decisionsToConfig(decisions []shape.Decision) map[string]shape.AliasValue {
	out := make(map[string]shape.AliasValue, len(decisions))
	for _, d := range decisions {
		if d.Tie {
			out[d.Path] = shape.AliasValue{Tie: append([]string(nil), d.TieCandidate...)}
			continue
		}
		out[d.Path] = shape.AliasValue{Resolved: d.WantAlias}
	}
	return out
}

func findModuleRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		} else if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("stat go.mod in %s: %w", dir, err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s upward", start)
		}
		dir = parent
	}
}

func printPackageErrors(pkgs []*packages.Package) int {
	count := 0
	for _, pkg := range pkgs {
		for _, err := range pkg.Errors {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			count++
		}
	}
	return count
}

func reportDecisions(fset *token.FileSet, decisions []shape.Decision) bool {
	found := false
	for _, d := range decisions {
		if d.Tie {
			fmt.Fprintf(os.Stdout, "%s: import %q has unresolved tie among aliases %s\n",
				d.Package, d.Path, aliasListDisplay(d.TieCandidate))
			found = true
			continue
		}
		for _, o := range d.Inconsistent {
			fmt.Fprintf(os.Stdout, "%s: import %q should use alias %s, not %s (package-wide majority)\n",
				fset.Position(o.Pos), d.Path, aliasDisplay(d.WantAlias), aliasDisplay(o.Alias))
			found = true
		}
	}
	return found
}

func reportCollisions(fset *token.FileSet, collisions []shape.AliasCollision) bool {
	found := false
	for _, c := range collisions {
		if len(c.Occurrences) == 0 {
			continue
		}
		fmt.Fprintf(os.Stdout, "%s: alias %q is used for multiple import paths in package %s\n",
			fset.Position(c.Occurrences[0].Pos), c.Alias, c.Package)
		found = true
	}
	return found
}

func reportDuplicates(fset *token.FileSet, duplicates []shape.DuplicateImport) bool {
	found := false
	for _, dup := range duplicates {
		for _, o := range dup.Occurrences {
			fmt.Fprintf(os.Stdout, "%s: import %q is imported multiple times in this file with different aliases\n",
				fset.Position(o.Pos), dup.Path)
			found = true
		}
	}
	return found
}

func aliasDisplay(alias string) string {
	if alias == "" {
		return "no alias"
	}
	return fmt.Sprintf("%q", alias)
}

func aliasListDisplay(aliases []string) string {
	if len(aliases) == 0 {
		return "[]"
	}
	parts := make([]string, len(aliases))
	for i, alias := range aliases {
		parts[i] = aliasDisplay(alias)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
