# Security

## Reporting a vulnerability

Please report suspected vulnerabilities **privately** by emailing the
maintainer rather than opening a public issue. Include a minimal
reproduction and the affected version (output of `100x --version`).

Expect an acknowledgement within seven days.

## Threat model

- API secrets are loaded into memory only for the duration of one
  process invocation; they are stored at rest in the OS keychain
  (`zalando/go-keyring`) or, when no keychain is available, in
  `$XDG_CONFIG_HOME/100x/credentials/<profile>` with mode 0600.
- Profile config (`$XDG_CONFIG_HOME/100x/config.toml`) holds only
  non-secret fields: client ID, env label, and environment settings.
- Requests are signed with HMAC-SHA256 over a fixed template
  (`client_id={cid}&nonce={nonce}&ts={ts}`). The nonce is 16 random
  bytes per request and the ts skew tolerance is ±10 seconds.

Out of scope: protection of the local environment after the user has
authenticated. Anyone with read access to the keychain or the
credentials file can act as the user.
