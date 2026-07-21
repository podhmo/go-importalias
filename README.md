# go-importalias

`go-importalias` checks whether imports that use the same package path are
written with a consistent alias inside a Go package, and provides automatic
fixes for safe cases.

The main entry point is the `goimportalias` binary. It can run as a `go vet`
vettool for analyzer-based checks, and also contains the standalone CLI mode
used for config generation and source rewrites.

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

The standalone CLI uses flat flags and accepts target packages as positional
arguments, for example `./...`.

| Flag | Default | Description |
|---|---|---|
| `-fix` | `false` | Apply safe automatic fixes. |
| `-config` | module-root `importalias.json` | Path to the configuration file. |
| `-strict` | `false` | Treat majority-based decisions strictly. |
| `-skip-generated` | `true` | Skip files marked with the standard generated-code comment. |

Exit codes:

- `0`: no inconsistencies remain, or all detected inconsistencies were fixed.
- `1`: inconsistencies remain, including unresolved ties.
- `2`: runtime error, such as I/O, invalid config, or package loading failure.

## Configuration

`importalias.json` stores per-package import alias decisions. A value is either
a string alias, an empty string for no explicit alias, or an array of two or
more aliases when a majority tie is unresolved.

```json
{
  "packages": {
    "*": {
      "github.com/example/project/foo": "foo"
    },
    "github.com/example/app/...": {
      "github.com/example/project/bar": ["bar", "barv2"]
    },
    "github.com/example/app/internal/api": {
      "github.com/example/project/baz": ""
    }
  },
  "ignore": [
    "github.com/example/app/generated/..."
  ]
}
```

Package scopes are checked in this order: exact package, longest `...` prefix,
then `*`.

## golangci-lint integration

The root package exports the analyzer as `importalias.Analyzer`, so tools that
embed `analysis.Analyzer` values, including custom golangci-lint integrations,
can wire it in directly.

## License

MIT
