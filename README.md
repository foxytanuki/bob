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

- `bob open <url>`
- `bob doctor`
- `bobd serve`
- `bobd init`

Current limitations:

- SSH tunnel creation is out of scope for MVP
- duplicate suppression is not implemented yet
- default policy is localhost-only

## Build

```bash
make build
```

Or:

```bash
go build ./...
```

## Quick start

1. Generate a shared token:

```bash
bobd init
```

2. On the local machine:

```bash
export BOBD_TOKEN=...
bobd serve
```

3. Create a port forward so the remote machine can reach local `bobd`

Example:

```bash
ssh -R 17331:127.0.0.1:7331 user@remote-host
```

4. On the remote machine:

```bash
export BOB_ENDPOINT=http://127.0.0.1:17331
export BOB_TOKEN=...
bob doctor
bob open http://127.0.0.1:5173
```

Important:

- `BOB_ENDPOINT` only points to `bobd`.
- The target app URL itself must already be reachable from the local machine.
- If your app is running remotely on `127.0.0.1:5173`, you usually need a **separate port forward** for that app too.

If automatic opening fails, `bob open` prints the URL so the user can open it manually.

## Environment

### Remote CLI

- `BOB_ENDPOINT` default: `http://127.0.0.1:17331`
- `BOB_TOKEN`
- `BOB_TIMEOUT` default: `5s`

### Local daemon

- `BOBD_BIND` default: `127.0.0.1:7331`
- `BOBD_TOKEN` required
- `BOBD_LOCALHOST_ONLY` default: `true`

## Docs

- [PLAN.md](./PLAN.md)
- [docs/protocol.md](./docs/protocol.md)
- [docs/setup-ssh.md](./docs/setup-ssh.md)
- [docs/security.md](./docs/security.md)
