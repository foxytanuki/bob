# SSH setup

MVP can work in two ways:

1. manual SSH forwarding
2. `bob tunnel` managing an SSH session for you

## Terms

- `bobd bind address`: where the local daemon listens, e.g. `127.0.0.1:7331`
- `forwarded endpoint`: the URL that remote `bob` can reach, e.g. `http://127.0.0.1:17331`

## Example flow

### Option A: manual SSH

1. Start local `bobd`

```bash
export BOBD_TOKEN=...
bobd serve
```

2. Create a port forward that makes local `127.0.0.1:7331` reachable as remote `127.0.0.1:17331`

One common pattern is starting SSH from the local machine with a reverse forward:

```bash
ssh -R 17331:127.0.0.1:7331 user@remote-host
```

3. On the remote machine:

```bash
export BOB_ENDPOINT=http://127.0.0.1:17331
export BOB_TOKEN=...
bob doctor
bob open http://127.0.0.1:5173
```

Important:

- This `BOB_ENDPOINT` forward is only for the `bob -> bobd` control request.
- `bob` does not rewrite app URLs.
- If the target app is remote-only, make it reachable locally with a separate port forward before calling `bob open`.

If your environment already provides another path from remote to local `bobd`, set `BOB_ENDPOINT` to that URL instead.

### Option B: `bob tunnel`

On the local machine:

```bash
export BOBD_TOKEN=...
bobd serve
bob tunnel up devbox --ssh user@remote-host --mirror 5173
```

This creates:

- remote `127.0.0.1:17331` -> local `127.0.0.1:7331` for `bob -> bobd`
- local `127.0.0.1:5173` -> remote `127.0.0.1:5173` for browser -> app

Then on the remote machine:

```bash
export BOB_ENDPOINT=http://127.0.0.1:17331
export BOB_TOKEN=...
bob doctor
bob open http://127.0.0.1:5173
```

To inspect or stop the tunnel later:

```bash
bob tunnel status devbox
bob tunnel down devbox
```
