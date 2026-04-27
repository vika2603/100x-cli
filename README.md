# 100x-cli

Command-line client for the 100x futures-trading API.

## Install

```sh
go install github.com/vika2603/100x-cli/cmd/100x@latest
```

Or build from source:

```sh
make build       # → ./bin/100x
make install     # → $GOBIN/100x
```

## Quick start

Add a profile (interactive secret prompt; the secret is stored in the OS
keychain, falling back to a chmod-600 file under `$XDG_CONFIG_HOME/100x/`).
Profiles hold user credentials. The API endpoint is built into the CLI; use
`E100X_ENDPOINT` only when you need to override it for one process:

```sh
100x profile add test --client-id <CID>
E100X_ENDPOINT=https://api.example.com 100x futures market state BTCUSDT
```

Then call any command:

```sh
100x futures balance list
100x futures market state BTCUSDT
100x futures order place BTCUSDT --side buy --price 70000 --size 0.1
```

Use `--json` for machine output, `--jq '<expr>'` to filter, `-q` to suppress
non-essential text:

```sh
100x --json futures order list BTCUSDT --jq 'map({order_id, side, price, volume, status})'
```

## Layout

```
api/futures/             # exported SDK; mirrors the gateway @server groups
api/futures/fake/        # in-memory Doer for tests and local dev
api/internal/transport/  # signed HTTP client (HMAC-SHA256, req library)
cmd/100x/                # entry, err → exit-code mapping
internal/cmd/            # cobra command tree (factory-injected, gh-style)
internal/{config,credential,output,exit,prompt,version}/
```

## Local dev without credentials

```sh
E100X_FAKE=1 100x futures balance list
```

The fake satisfies `futures.Doer`; reads return canned shapes and writes
update an in-memory map so `place` → `list` → `cancel` round-trips work
within a single process.

## Tests

```sh
make test        # go test -race -count=1 ./...
make lint        # golangci-lint run (requires golangci-lint on PATH)
```

## Releasing

Tagging `vX.Y.Z` and pushing the tag triggers `.github/workflows/release.yml`,
which runs `goreleaser` to build cross-platform binaries (Linux / macOS /
Windows × amd64 / arm64), publish a GitHub release, and attach a
`checksums.txt`.

For a local dry run:

```sh
make snapshot      # builds ./dist/ without publishing
make release-check # validate .goreleaser.yaml
```

## License

MIT — see [LICENSE](LICENSE).
