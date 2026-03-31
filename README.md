# bob

Remote→Local browser open bridge.

## What it is

`bob` is a generic helper for remote development workflows where a process running on a remote machine wants to open a URL in the user's local browser.

Examples:

- Vite dev server on SSH/devcontainer
- coding agents that launch local review UIs
- docs preview servers
- internal dashboards and local tools

Instead of trying to open a browser on the remote machine, a remote CLI forwards an `open_url` request to a local daemon, which validates it and opens the user's browser locally.

## MVP shape

```text
Remote app -> bob CLI -> SSH tunnel -> bobd local daemon -> local browser
```

## Components

- `bob`: remote bridge CLI
- `bobd`: local daemon
- transport: SSH first, Tailscale/relay later

## Goals

- app-agnostic
- SSH/devcontainer friendly
- safe fallback when the daemon is unavailable
- no cloud dependency for MVP

## Non-goals

- app-specific integrations first
- relay/SaaS first
- full UI/admin surface first

## Planned behavior

1. Remote tool calls `bob open <url>`
2. `bob` sends an authenticated request to `bobd`
3. `bobd` validates the request and opens the local browser
4. If the request fails, `bob` prints the URL so the user can open it manually

## Docs

- [PLAN.md](./PLAN.md)
