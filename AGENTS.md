# AGENTS.md

## Verification commands

Use these commands directly. Do not look for a `Makefile`, scripts, or `.golangci.yml`.

- Test: `go test ./...`
- Lint: `go vet ./...` (no extra linters such as `golangci-lint`)
- Build (when needed): `go build ./...`
- Format: run `gofmt -l -w .` anytime; it is **required before every commit**.

Only commit or open a PR when explicitly asked.

## Issue workflow (docs/issues/)

Full rules: ADR-01 (`docs/adr/01-issue-workflow.md`) and ADR-02
(`docs/adr/02-issue-impl-loop-commands.md`). Summary to avoid re-deriving
this every time:

- Pick one issue from the open table in `docs/issues/README.md`.
- "Done" includes closing the issue, not just green tests. Once every exit
  condition in the issue file is met: `git mv` it into `docs/issues/closed/`
  and delete its row from the open table in `docs/issues/README.md`.
- Fold the close step into the same commit as the implementation unless told
  otherwise. Precedent (issue 01, closed in isolation once) is a plain
  rename — don't add completion-date/commit metadata to the closed file.
