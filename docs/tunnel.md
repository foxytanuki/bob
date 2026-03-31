# Tunnel

`bob tunnel` manages the SSH forwarding needed by `bob open`.

## Commands

```bash
bob tunnel up <name> --ssh <target> [--mirror <port>]...
bob tunnel status [<name>|--all]
bob tunnel down <name>
```

## Example

Local machine:

```bash
export BOBD_TOKEN=...
bobd serve
bob tunnel up devbox --ssh user@remote-host --mirror 8787
```

Remote machine:

```bash
export BOB_ENDPOINT=http://127.0.0.1:17331
export BOB_TOKEN=...
bob doctor
bob open http://127.0.0.1:8787
```

## What `--mirror` does

`--mirror 8787` means:

- local `127.0.0.1:8787` forwards to remote `127.0.0.1:8787`

That matters because `bob open` sends the raw URL to local `bobd`, and local browser then tries to open the same URL.

## Notes

- `bob tunnel` uses the system `ssh` command.
- tunnel state is stored under XDG state dir, e.g. `~/.local/state/bob/`.
- tokens are **not** stored in tunnel state.
- MVP supports same-port mirroring only; URL rewrite is not implemented.
