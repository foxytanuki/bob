# Protocol

## Endpoint

- `GET /healthz`
- `POST /open`

`bob` talks to a **forwarded endpoint**: a URL on the remote side that already reaches local `bobd` through an SSH port forward or equivalent transport.

Example:

```text
remote bob  ->  http://127.0.0.1:17331/open
forwarded   ->  local bobd 127.0.0.1:7331
```

## `POST /open`

Headers:

- `Content-Type: application/json`
- `Authorization: Bearer <token>`

Request body:

```json
{
  "version": 1,
  "action": "open_url",
  "url": "http://127.0.0.1:5173",
  "source": {
    "app": "bob",
    "host": "devbox",
    "cwd": "/workspace/app"
  },
  "timestamp": 1712345678,
  "nonce": "random-id"
}
```

### Success

```json
{
  "ok": true,
  "status": "OPENED"
}
```

### Failure statuses

- `INVALID_REQUEST`
- `INVALID_URL`
- `UNAUTHORIZED`
- `DENIED`
- `INTERNAL_ERROR`

## Concurrency

- `bobd` may accept multiple requests at the same time.
- Requests are treated independently.
- No duplicate suppression is implemented in MVP.
