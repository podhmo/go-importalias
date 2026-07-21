# AGENTS.md

## Verification commands

Use these commands directly. Do not look for a `Makefile`, scripts, or `.golangci.yml`.

- Test: `go test ./...`
- Lint: `go vet ./...` (no extra linters such as `golangci-lint`)
- Build (when needed): `go build ./...`
- Format: run `gofmt -l -w .` anytime; it is **required before every commit**.

Only commit or open a PR when explicitly asked.
