# 100x-cli

Command-line client for the 100x futures-trading API.

## Install

One-liner (Linux / macOS, amd64 / arm64). Downloads the latest release
tarball, verifies the SHA-256, and drops the binary in `$HOME/.local/bin`:

```sh
curl -fsSL https://raw.githubusercontent.com/vika2603/100x-cli/main/script/install.sh | sh
```

Pin a version or pick a directory:

```sh
curl -fsSL https://raw.githubusercontent.com/vika2603/100x-cli/main/script/install.sh \
  | sh -s -- --version v0.3.0 --to /usr/local/bin
```

Or with Go:

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
Profiles hold the client ID; secrets are stored separately:

```sh
100x profile add test --client-id <CID>
100x futures market state BTCUSDT
```

Set `$E100X_ENDPOINT` to point one process at a different API host:

```sh
E100X_ENDPOINT=https://api.example.com 100x futures market state BTCUSDT
```

### Building from source

Set `E100X_ENDPOINT_DEFAULT` so the resulting binary has a default endpoint
and end users do not need `$E100X_ENDPOINT`:

```sh
E100X_ENDPOINT_DEFAULT=https://api.example.com make build
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
