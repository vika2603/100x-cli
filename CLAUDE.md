# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Go 1.25 CLI for the 100x crypto futures exchange. Single binary `100x` built from `cmd/100x/main.go`, which delegates to `internal/cmd/root`. Uses cobra for CLI structure and charm (bubbletea, huh, lipgloss) for interactive TUI prompts.

## Developer commands

| Command | What it does |
|---|---|
| `make build` | Build to `bin/100x` (bakes version/commit via ldflags) |
| `make install` | `go install ./cmd/100x` |
| `make test` | `go test -race -count=1 ./...` |
| `make lint` | `golangci-lint run` (requires golangci-lint v2 on PATH) |
| `make fmt` | `gofmt -s -w .` then `goimports -w .` |
| `make vet` | `go vet ./...` |
| `make tidy` | `go mod tidy` |
| `make run ARGS="..."` | Build then run with args |
| `make snapshot` | Local goreleaser build → `dist/` |
| `make release-check` | Validate `.goreleaser.yaml` |

CI order (on push/PR): `vet → build → test -race → lint → govulncheck`. Use `/verify` to run the same chain locally before declaring a task done.

## Architecture

```
cmd/100x/main.go          — entrypoint; signal handling → root.NewCmdRoot()
internal/cmd/             — cobra subcommands: root, profile, futures, market, upgrade, completion
api/futures/              — exchange API client (orders, positions, market, asset, settings)
api/internal/             — shared HTTP transport
internal/config/          — TOML profile loader + XDG paths
internal/credential/      — OS keychain (zalando/go-keyring) with file fallback
internal/output/          — table rendering + --json / --jq formatting
internal/prompt/          — huh-based interactive prompts for secrets
internal/session/         — per-request auth/session state
internal/timeexpr/        — natural-language time parsing
internal/clierr/          — error classification
internal/exit/            — exit-code mapping
internal/version/         — ldflags-populated version info
internal/mocks/           — go:generate mock (uber/mock) for futures.Doer
```

## Config & credentials

- **Config file**: `$XDG_CONFIG_HOME/100x/config.toml` (default `~/.config/100x/config.toml`)
- **Credentials dir**: `$XDG_CONFIG_HOME/100x/credentials/` (file fallback, chmod 600)
- Secrets are stored in the OS keychain when available; credential files are the fallback.
- `--profile <name>` selects which profile's credentials to use.

## Environment variables

| Variable | Purpose |
|---|---|
| `E100X_ENDPOINT` | Override API endpoint at runtime |
| `E100X_ENDPOINT_DEFAULT` | Build-time default endpoint (baked via ldflags) |
| `E100X_PROFILE` | Default profile name (mise sets `test`) |
| `XDG_CONFIG_HOME` | Config directory base (default `~/.config`) |

`mise.toml` pins Go 1.25 and sets `E100X_PROFILE=test`, `E100X_ENDPOINT=https://api.lyantechinnovation.com/`. Run `mise install` before first dev session if using mise.

## Testing

- Run all: `make test`
- Single test: `go test -race -count=1 -run TestName ./path/to/pkg`. Always keep `-race -count=1` to match CI.
- Mocks: regenerate with `go generate ./...` (uses `go.uber.org/mock`). Do not hand-edit files in `internal/mocks/`.
- Linter exclusions: `gosec` / `errcheck` disabled on `_test.go`; `gocritic typeSwitchVar` disabled on `api/futures/fake/`.

## Tooling

- **mise**: `mise.toml` pins Go 1.25 and dev env vars.
- **golangci-lint v2**: config in `.golangci.yml`. CI pins `v2.11.4`. v1 will fail with config-version errors.
- **goimports** local-prefix: `github.com/vika2603/100x-cli`. The Makefile's `goimports -w .` does not pass `-local`, but the lint config does — run `make fmt` (not just `gofmt`) to match the formatter the project actually uses.
- **goreleaser**: `make snapshot` for local builds; `v*` tag triggers release workflow.

## Releases

- Triggered by pushing `v*` tags.
- goreleaser builds for linux/darwin/windows × amd64/arm64 (no windows/arm64).
- `E100X_ENDPOINT_DEFAULT` is set from a GitHub repo variable during release.

## Claude-specific notes

- The repo's day-to-day CLI workflows (auth, market, portfolio, orders, positions, errors) are documented in `skills/100x-cli/SKILL.md` with reference docs under `skills/100x-cli/references/`. Consult that skill before running any `100x` subcommand on the user's behalf.
- `--size` on order commands is the **exchange quantity**, not USDT notional. Do not convert.
- After editing Go code, run `/verify` (or `make vet build test lint` in CI order) before declaring a task done. If `make lint` complains about formatting / imports, the fix is `make fmt`, not hand-editing.
