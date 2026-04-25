# AGENTS.md

Guidance for AI coding agents working on this repository.

## Project overview

`bob` is a Remote→Local browser open bridge written in Go.

- `bob`: remote-side CLI. Sends URL open requests to a reachable `bobd` endpoint.
- `bobd`: local-side daemon. Authenticates requests, validates policy, and opens the local browser.
- Transport is currently SSH reverse forwarding / tunnel based.

Typical flow:

```text
remote dev server/tool -> bob -> SSH-forwarded endpoint -> bobd -> local browser
```

## Important concepts

- `endpoint` / `BOB_ENDPOINT`: URL that remote `bob` uses to reach `bobd`, usually `http://127.0.0.1:17331` on the remote machine via SSH reverse forwarding.
- `session` / `BOB_SESSION`: tunnel session name. It should match `bobd serve --tunnel-name <name>` or `bob tunnel up <name>`. Required for remote loopback URLs such as `http://127.0.0.1:5173`.
- Token: bearer token shared between `bob` and `bobd`.
- Config files live under `$XDG_CONFIG_HOME/bob/`, falling back to `~/.config/bob/`.

## Commands

Build and test:

```bash
go test ./...
just build
```

Initialize local daemon config:

```bash
bobd init
bobd init --force
```

Initialize remote CLI config:

```bash
bob init --token <token> --session <name>
```

Run daemon with tunnel creation:

```bash
bobd serve --tunnel-name devbox --ssh user@remote-host
```

## Config behavior

- Config file priority is: environment variable > JSON config file > hardcoded default.
- `bob init` writes `bob.json` with `0600` permissions and refuses overwrite unless `--force` is used.
- `bobd init` writes `bobd.json` with `0600` permissions and refuses overwrite unless `--force` is used.
- Tests that touch config MUST set `XDG_CONFIG_HOME=t.TempDir()` to avoid reading or modifying a developer's real config.

## Repository notes

- Keep dependencies minimal; the project currently uses the Go standard library only.
- Prefer small, focused changes with tests.
- Run `gofmt` on changed Go files.
- Do not commit generated binaries or local config files.
- OpenSpec changes live under `openspec/changes/`; update artifacts when behavior/requirements change.
