# Protocol

## Endpoints

- `GET /healthz`
- `POST /open` (legacy v1)
- `POST /v2/open` (current)

`bob` talks to a **forwarded endpoint**: a URL on the remote side that already reaches local `bobd` through an SSH port forward or equivalent transport.

Example:

```text
remote bob  ->  http://127.0.0.1:17331/v2/open
forwarded   ->  local bobd 127.0.0.1:7331
```

## `POST /open` (v1)

Legacy endpoint kept for compatibility.

```json
{
  "version": 1,
  "action": "open_url",
  "url": "http://127.0.0.1:5173"
}
```

It opens the raw URL unchanged.

## `POST /v2/open`

Headers:

- `Content-Type: application/json`
- `Authorization: Bearer <token>`

Request body:

```json
{
  "version": 2,
  "action": "open_url",
  "session": "devbox",
  "url": "http://127.0.0.1:5173",
  "source": {
    "app": "bob",
    "host": "remote-host",
    "cwd": "/workspace/app"
  },
  "timestamp": 1712345678,
  "nonce": "random-id"
}
```

### Session rules

- `session` is required for **loopback** URLs that may need auto-mirror
- `session` may be empty for non-loopback URLs

### Success

```json
{
  "ok": true,
  "status": "OPENED",
  "opened_url": "http://127.0.0.1:43123",
  "rewritten": true,
  "local_port": 43123,
  "mapping_reused": false
}
```

If no rewrite was needed, `opened_url` may be omitted.

### Failure statuses

- `INVALID_REQUEST`
- `INVALID_URL`
- `UNAUTHORIZED`
- `DENIED`
- `SESSION_REQUIRED`
- `SESSION_NOT_FOUND`
- `MIRROR_FAILED`
- `INTERNAL_ERROR`

## Concurrency

- `bobd` may accept multiple requests at the same time.
- mirror creation is serialized per session.
- repeated opens for the same session/remote port reuse the stored local mapping.
