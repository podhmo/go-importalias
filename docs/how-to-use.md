# How to use go-importalias

## Installation

```sh
go install github.com/podhmo/go-importalias/cmd/goimportalias@latest
```

## Use with go vet

After installation, run the analyzer through `go vet`:

```sh
go vet -vettool=$(which goimportalias) ./...
```

Vet mode is read-only: it reports import alias inconsistencies and never writes
configuration files or source files.

## Standalone CLI

The standalone CLI accepts target packages as positional arguments. When no
package pattern is supplied, it uses `./...`.

```sh
goimportalias ./...
goimportalias -fix ./...
goimportalias -config ./importalias.json ./...
```

| Flag | Default | Description |
|---|---|---|
| `-fix` | `false` | Apply safe automatic fixes. |
| `-config` | module-root `importalias.json` | Path to the configuration file. |
| `-strict` | `false` | When generating config, keep all observed alias candidates instead of collapsing a majority. Cannot be combined with `-fix`. |
| `-skip-generated` | `true` | Skip files marked with the standard generated-code comment. |

Exit codes:

- `0`: no inconsistencies remain, or all detected inconsistencies were fixed.
- `1`: inconsistencies remain, including unresolved ties.
- `2`: runtime error, such as I/O, invalid config, or package loading failure.

## Configuration

`importalias.json` stores per-package import alias decisions. A value is either:

- a string alias;
- an empty string (`""`) meaning no explicit alias; or
- an array of two or more aliases when candidates should remain unresolved (for
  example a tie, or `-strict` config generation).

When the CLI generates or updates config, it omits entries whose only decision is
`""` because the normal no-alias case does not need to be written down. This
keeps `importalias.json` small. If no-alias is one of multiple unresolved
candidates, it is kept in the candidate array, for example `["", "foo"]`.

```json
{
  "packages": {
    "*": {
      "github.com/example/project/foo": "foo"
    },
    "github.com/example/app/...": {
      "github.com/example/project/bar": ["bar", "barv2"],
      "github.com/example/project/baz": ["", "baz"]
    }
  },
  "ignore": [
    "github.com/example/app/generated/..."
  ]
}
```

Package scopes are checked in this order: exact package, longest `...` prefix,
then `*`.

### ignore

The top-level `ignore` array excludes packages from both detection and `-fix`.
Each entry can be:

- an exact package path, such as `github.com/example/app/internal/legacy`;
- a `...` prefix pattern, such as `github.com/example/app/generated/...`; or
- `*`, which matches every package.

`ignore` is independent from `-skip-generated`: use `ignore` for package-level
exclusions, and `-skip-generated=false` only when generated files should be
included in scanning.
