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
Profiles hold the client ID; secrets are stored separately. The API endpoint is
built into the CLI; use `E100X_ENDPOINT` only when you need to override it for
one process:

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

Use `--json` for machine output, `--jq '<expr>'` to enable and filter that
JSON, and `-q` to suppress non-essential text. JSON keeps the API field names;
human labels may be friendlier:

```sh
100x --json futures order list --symbol BTCUSDT --jq 'map({order_id, side, price, volume, status})'
```

## License

MIT — see [LICENSE](LICENSE).
