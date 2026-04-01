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
- `bob open <url>`
- `bob doctor`
- `bob tunnel up/status/down`
- `bobd serve`
- `bobd init`

Current limitations:

- `bob open` still does not create the first SSH session from the remote side
- duplicate suppression is not implemented yet
- default policy is localhost-only

## Build

```bash
just build
```

Version を埋めてビルド:

```bash
VERSION=v0.1.0 just build
```

Install to `~/.local/bin`:

```bash
just install
```

Custom install dir:

```bash
just install BINDIR=/custom/bin
```

Or:

```bash
go build ./...
```

CLI 版番号確認:

```bash
bob version
bobd version
```

## Versioning

- Git tag は `vX.Y.Z` 形式
- アプリ版番号は SemVer を採用
- 開発中ビルドのデフォルト値は `dev`

例:

```bash
VERSION=v0.1.0 just build-binaries
git tag v0.1.0
```

必要なら commit/date も埋め込めます:

```bash
VERSION=v0.1.0 COMMIT=$(git rev-parse --short HEAD) DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) just build-binaries
```

## Quick start

1. Generate a shared token:

```bash
bobd init
```

2. On the local machine:

```bash
export BOBD_TOKEN=...
bobd serve --tunnel-name devbox --ssh user@remote-host
```

3. On the remote machine:

```bash
export BOB_ENDPOINT=http://127.0.0.1:17331
export BOB_TOKEN=...
export BOB_SESSION=devbox
bob doctor
bob open http://127.0.0.1:5173
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
- loopback app URLs can now be mirrored automatically after the control tunnel exists.
- if the same local port is busy, `bobd` may allocate another local port and rewrite the opened URL.

If automatic opening fails, `bob open` prints the URL so the user can open it manually.

## Environment

### Remote CLI

- `BOB_ENDPOINT` default: `http://127.0.0.1:17331`
- `BOB_TOKEN`
- `BOB_SESSION` required for auto-mirror, set to the tunnel name
- `BOB_TIMEOUT` default: `5s`

### Local daemon

- `BOBD_BIND` default: `127.0.0.1:7331`
- `BOBD_TOKEN` required
- `BOBD_LOCALHOST_ONLY` default: `true`

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
