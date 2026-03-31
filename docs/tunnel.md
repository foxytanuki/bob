# Tunnel

`bob tunnel` manages the SSH forwarding needed by `bob open`.

## Commands

```bash
bob tunnel up <name> --ssh <target> [--mirror <port>]...
bob tunnel status [<name>|--all]
bob tunnel down <name>
```

`--mirror` is now optional. The recommended path is:

1. start the control tunnel either with `bobd serve --tunnel-name ... --ssh ...` or explicitly with `bob tunnel up`
2. let `bobd` auto-create loopback app mirrors on demand when `bob open` runs

## Example

Local machine:

```bash
export BOBD_TOKEN=...
bobd serve --tunnel-name devbox --ssh user@remote-host
```

Remote machine:

```bash
export BOB_ENDPOINT=http://127.0.0.1:17331
export BOB_TOKEN=...
export BOB_SESSION=devbox
bob doctor
bob open http://127.0.0.1:8787
```

If local `127.0.0.1:8787` is free, `bobd` can mirror that same port. If it is busy, `bobd` may allocate a different local port and open that rewritten URL instead.

## What `--mirror` still does

`--mirror 8787` means:

- local `127.0.0.1:8787` forwards to remote `127.0.0.1:8787`

That matters when you want to pin a mirror up front instead of letting `bobd` allocate one on demand.

## Notes

- `bob tunnel` uses the system `ssh` command.
- `bobd serve --tunnel-name ... --ssh ...` can bootstrap the initial control tunnel and shuts it down when `bobd` exits.
- tunnel state is stored under XDG state dir, e.g. `~/.local/state/bob/`.
- tokens are **not** stored in tunnel state.
- `bob open` auto-mirror requires `BOB_SESSION`.
