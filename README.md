# bob

Remote→Local browser open bridge.

## What it is

`bob` lets a process running on a remote machine ask your local machine to open a URL in your local browser.

MVP flow:

```text
Remote app/tool -> bob CLI -> forwarded endpoint -> bobd local daemon -> local browser
```

Where:

- `bob` is the remote CLI
- `bobd` is the local daemon
- `forwarded endpoint` is the URL visible from the remote side that already reaches local `bobd`

Example:

```text
local bobd listens on 127.0.0.1:7331
SSH makes that reachable on the remote side as 127.0.0.1:17331
remote bob sends POST http://127.0.0.1:17331/open
```

## Status

This repository now contains a minimal Go MVP scaffold:

- `bob <url>`
- `bob init`
- `bob open <url>`
- `bob code-server [path]`
- `bob doctor`
- `bob tunnel up/status/down`
- `bobd serve`
- `bobd init`

Current limitations:

- `bob open` still does not create the first SSH session from the remote side
- duplicate suppression is not implemented yet
- default policy is localhost-only

## Build

Build with the default version:

```bash
just build
```

Build with an explicit version:

```bash
VERSION=v0.5.0 just build
```

Install to `~/.local/bin`:

```bash
just install
```

Install to a custom directory:

```bash
just install BINDIR=/custom/bin
```

Or build directly with Go:

```bash
go build ./...
```

Check CLI versions:

```bash
bob version
bobd version
```

## Versioning

- Git tags use the `vX.Y.Z` format.
- Application versions follow SemVer.
- Repository builds default to `v0.5.0`.

Example release build:

```bash
VERSION=v0.5.0 just build-binaries
git tag v0.5.0
```

Include commit and build date if needed:

```bash
VERSION=v0.5.0 COMMIT=$(git rev-parse --short HEAD) DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) just build-binaries
```

## Quick start

The common setup is:

- local machine: run `bobd` and open the browser
- remote machine: run `bob` from dev tools / shells
- SSH reverse forwarding makes remote `127.0.0.1:17331` reach local `bobd`

1. Generate a shared token and local daemon config:

```bash
bobd init
```

This writes the local daemon config to `~/.config/bob/bobd.json` (or `$XDG_CONFIG_HOME/bob/bobd.json`) with `0600` permissions and prints a remote `bob.json` snippet.

2. On the local machine:

```bash
bobd serve --tunnel-name devbox --ssh user@remote-host
```

3. On the remote machine, initialize `bob` using the token printed by `bobd init`:

```bash
bob init --token <shared-token> --session devbox
```

This writes `~/.config/bob/bob.json` with `0600` permissions. You can also pass `--endpoint`, `--timeout`, and `--force`:

```bash
bob init --token <shared-token> --endpoint http://127.0.0.1:17331 --session devbox --timeout 5s
```

Alternatively, create `~/.config/bob/bob.json` manually using the snippet printed by `bobd init`:

```json
{
  "endpoint": "http://127.0.0.1:17331",
  "token": "<shared-token>",
  "session": "devbox"
}
```

Then verify and open URLs:

```bash
bob doctor
bob open http://127.0.0.1:5173
```

### Open code-server

`bob code-server [path]` opens an already-running remote code-server instance in your local browser through the existing loopback mirror flow. It does not start code-server.

Run code-server on the remote machine bound to loopback, for example `127.0.0.1:8080`, then run:

```bash
bob code-server .
bob code-server ~/repo --port 65508
```

The command resolves the folder to an absolute path, builds `http://127.0.0.1:<port>/?folder=<path>`, and sends it through the normal `bob open` `/v2/open` flow. Because the URL is loopback, `session` / `BOB_SESSION` must match an active tunnel session so `bobd` can mirror the remote port locally.

If `~/.config/bob/bob.json` already exists, `bob init` refuses to overwrite it. To intentionally update it, delete the config file first or run `bob init --force --token <shared-token> --session <name> ...`.

If `~/.config/bob/bobd.json` already exists, `bobd init` refuses to overwrite it to avoid accidentally rotating the token and breaking remote machines. To intentionally regenerate the token, delete the config file first or run:

```bash
bobd init --force
```

If you prefer to keep the SSH session separate, you can still create the port forward explicitly.

Manual SSH example:

```bash
ssh -R 17331:127.0.0.1:7331 user@remote-host
```

Or via `bob tunnel` on the local machine:

```bash
bob tunnel up devbox --ssh user@remote-host
```

Important:

- `BOB_ENDPOINT` only points to `bobd`.
- `session` / `BOB_SESSION` should match the tunnel name, e.g. `devbox`.
- Loopback app URLs can be mirrored automatically after the control tunnel exists.
- If the same local port is busy, `bobd` may allocate another local port and rewrite the opened URL.

If automatic opening fails, `bob open` prints the URL so the user can open it manually.

## Configuration

`bob` and `bobd` read JSON config files from `$XDG_CONFIG_HOME/bob/`, falling back to `~/.config/bob/`.

Environment variables override config file values, which is useful for temporary overrides, CI, and containers.

### What endpoint and session mean

- `endpoint`: the remote-visible URL for the local daemon. With the default SSH tunnel this is `http://127.0.0.1:17331` on the remote machine.
- `session`: the tunnel name used to mirror remote loopback URLs back to the local browser. For remote-hosted dev servers, set this during init with `bob init --session <name>`.

### Remote CLI config

`~/.config/bob/bob.json`:

```json
{
  "endpoint": "http://127.0.0.1:17331",
  "token": "<shared-token>",
  "session": "devbox",
  "timeout": "5s",
  "codeServer": {
    "port": 8080
  }
}
```

### Local daemon config

`~/.config/bob/bobd.json`:

```json
{
  "bind": "127.0.0.1:7331",
  "token": "<shared-token>",
  "localhost_only": true
}
```

## Environment overrides

### Remote CLI

- `BOB_ENDPOINT` overrides `endpoint` (default: `http://127.0.0.1:17331`)
- `BOB_TOKEN` overrides `token`
- `BOB_SESSION` overrides `session`; required for auto-mirror, set to the tunnel name
- `BOB_TIMEOUT` overrides `timeout` (default: `5s`)
- `BOB_CODE_SERVER_PORT` overrides `codeServer.port` for `bob code-server` (default: `8080`)

### Local daemon

- `BOBD_BIND` overrides `bind` (default: `127.0.0.1:7331`)
- `BOBD_TOKEN` overrides `token` (required via env or config)
- `BOBD_LOCALHOST_ONLY` overrides `localhost_only` (default: `true`)

`bobd serve` flags:

- `--tunnel-name` tunnel session name to create on startup
- `--ssh` SSH target for that session
- `--remote-bob-port` remote loopback port for `BOB_ENDPOINT` (default: `17331`)
- `--local-bobd` local bobd address forwarded over SSH (default: `BOBD_BIND`)

## Docs

- [PLAN.md](./PLAN.md)
- [docs/protocol.md](./docs/protocol.md)
- [docs/setup-ssh.md](./docs/setup-ssh.md)
- [docs/tunnel.md](./docs/tunnel.md)
- [docs/security.md](./docs/security.md)
