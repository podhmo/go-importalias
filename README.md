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

```sh
go vet -vettool=$(which goimportalias) ./...
```

Vet mode is read-only: it reports import alias inconsistencies and never writes
configuration files or source files.

## Usage

See [docs/how-to-use.md](docs/how-to-use.md) for standalone CLI flags,
configuration, exit codes, and package ignore settings.

## golangci-lint integration

The root package exports the analyzer as `importalias.Analyzer`, so tools that
embed `analysis.Analyzer` values, including custom golangci-lint integrations,
can wire it in directly.

## License

MIT
