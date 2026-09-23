# How to use go-importalias

`go-importalias` has two entry points:

- `go vet` vettool mode for a quick read-only check.
- Standalone CLI mode for generating `importalias.json` and applying safe fixes.

Install the command first:

```sh
go install github.com/podhmo/go-importalias/cmd/goimportalias@latest
```

## Quick read-only check with go vet

Use `go vet` when you only want diagnostics:

```sh
go vet -vettool=$(which goimportalias) ./...
```

Vet mode does not create configuration files and does not rewrite source files.

## Scan and create configuration

Run the standalone command without `-fix` to scan packages and create or update
`importalias.json`:

```sh
goimportalias ./...
```

If `-config` is omitted, the file is written at the module root:

```sh
goimportalias -config ./importalias.json ./...
```

The generated configuration records non-empty alias decisions and unresolved
multiple-candidate decisions for each package. For example:

```json
{
  "packages": {
    "example.com/app": {
      "github.com/example/project/foo": "foo",
      "github.com/example/project/bar": ["", "barv2"]
    }
  }
}
```

Values mean:

- `"foo"`: use the explicit alias `foo`.
- `["", "barv2"]`: multiple candidates were found and no single alias was
  chosen automatically.
- `""`: no explicit alias. The config reader understands this value, and it can
  appear inside a multiple-candidate array, but generated resolved no-alias
  entries are omitted to keep `importalias.json` small.

`goimportalias` exits with status `1` when inconsistencies or unresolved
candidates remain, even after writing the configuration.

## Ignore packages

Add `ignore` to `importalias.json` when some packages should be skipped by both
scan and fix:

```json
{
  "ignore": [
    "example.com/app/generated/...",
    "example.com/app/internal/legacy"
  ],
  "packages": {
    "example.com/app": {
      "github.com/example/project/foo": "foo"
    }
  }
}
```

Ignore patterns use the same package-scope syntax as package decisions:

- `example.com/app/internal/legacy`: exact package.
- `example.com/app/generated/...`: that package and its subpackages.
- `*`: every package.

When the CLI updates `importalias.json`, existing `ignore` entries are preserved.

## Apply fixes

Run with `-fix` to rewrite source files when the desired alias is unambiguous:

```sh
goimportalias -fix ./...
```

The fix rewrites both the import declaration and the matching qualified
identifiers in the same file. For example, if `f` is the package-wide alias for
`fmt`, a file that imports plain `fmt` and calls `fmt.Println` can be rewritten
to import `f "fmt"` and call `f.Println`.

`-fix` only applies safe changes. It skips unresolved multiple-candidate entries
and cases where rewriting would collide with an existing local identifier. Files
that are already using the chosen alias are left unchanged.

After applying fixes, the command scans again. Exit status `0` means no
inconsistencies remain; exit status `1` means some entries still need manual
resolution.

## Resolve multiple candidates, then fix again

When the scan writes an array in `importalias.json`, choose one candidate
manually before rerunning `-fix`.

For example, change this unresolved entry:

```json
{
  "packages": {
    "example.com/app": {
      "github.com/example/project/bar": ["", "barv2"]
    }
  }
}
```

to a single chosen alias:

```json
{
  "packages": {
    "example.com/app": {
      "github.com/example/project/bar": ""
    }
  }
}
```

Then run the fixer again:

```sh
goimportalias -fix ./...
```

This second fix pass uses your configuration choice and rewrites safe remaining
uses to the selected alias.

## Useful flags

| Flag | Default | Description |
|---|---|---|
| `-fix` | `false` | Apply safe source rewrites. |
| `-config` | module-root `importalias.json` | Read and write configuration at a custom path. |
| `-strict` | `false` | Treat multiple observed aliases as unresolved instead of using a majority decision. |
| `-skip-generated` | `true` | Skip files with the standard generated-code marker. |
| `-include-tests` | `false` | Include `*_test.go` files in the scan. Loads test-variant packages, so internal test files join the package's majority vote and external `*_test` packages are scanned as packages of their own. |

Whether `*_test.go` files are analyzed defaults to each host tool's own
default: `go vet` analyzes them, so vet mode does too unless you pass
`-importalias.include_tests=false`; the standalone CLI follows `go/packages`
(`Tests: false`) and excludes them unless `-include-tests` is given.
