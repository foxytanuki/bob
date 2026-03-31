# Security

## MVP defaults

- bearer token required for `POST /open`
- only `http` and `https` URLs are accepted
- default policy is `localhost-only`
- daemon should bind to loopback unless you intentionally expose it elsewhere

## Logging

- `bobd` logs allow/deny decisions
- query strings are redacted in logs

## Not yet implemented

- nonce replay protection
- first-use approval
- duplicate suppression
- per-host allowlists beyond loopback mode
