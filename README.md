# 100x-cli

Command-line client for the 100x futures-trading API.

## Features

- Place, cancel, and list orders; attach SL / TP triggers
- View positions, balances, and live market state
- Manage multiple profiles; secrets stored in the OS keychain
- `--json` / `--jq` for scripting; human-readable tables by default
- Shell completion for bash, zsh, and fish

## Install

One-liner (Linux / macOS, amd64 / arm64):

```sh
curl -fsSL https://raw.githubusercontent.com/vika2603/100x-cli/main/script/install.sh | sh
```

Or with Go:

```sh
go install github.com/vika2603/100x-cli/cmd/100x@latest
```

Or build from source: `make build`.

## Quick start

```sh
100x profile add test --client-id <CID>      # interactive secret prompt
100x futures balance list
100x futures market state BTCUSDT
100x futures order place BTCUSDT --side buy --price 70000 --size 0.1
```

Run `100x --help` to see all commands.

## License

MIT — see [LICENSE](LICENSE).
