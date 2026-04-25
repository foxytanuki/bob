## Why

Currently `bob` and `bobd` rely entirely on environment variables for configuration. This forces users to repeatedly export variables in every shell session and loses configuration when shells restart. Moving to persistent XDG-compliant config files improves the developer experience while keeping environment variables available as overrides.

## What Changes

- Replace internal env-only loaders (`LoadCLIFromEnv()` / `LoadDaemonFromEnv()`) with file-aware loaders
- Add `internal/config/file.go` for XDG config directory resolution and JSON config loading
- `bobd init` will write the generated token to `~/.config/bob/bobd.json` instead of only printing shell exports
- `bobd init` will refuse to overwrite an existing daemon config by default, with an explicit force option for regeneration
- `bobd init` output will include remote-side `bob.json` guidance so users can configure the remote CLI without manually translating env vars
- Add `bob init` so the remote CLI can write its own `~/.config/bob/bob.json` from required token/session flags plus optional endpoint/timeout flags
- `bob` and `bobd` will read configuration from `~/.config/bob/bob.json` and `~/.config/bob/bobd.json` respectively
- Environment variables continue to override config file values for backward compatibility and CI/container use cases
- Config files created with `0o600` permissions since they contain authentication tokens

## Capabilities

### New Capabilities
- `xdg-config-loading`: File-based configuration loading with XDG Base Directory compliance and env var override

### Modified Capabilities
- None (no existing specs)

## Impact

- `internal/config/cli.go` — loader logic change
- `internal/config/daemon.go` — loader logic change
- `internal/config/file.go` — new file
- `internal/app/bobdapp/serve.go` — `bobd init` command behavior change
- `internal/app/bobdapp/usage.go` — document force option if added to `bobd init`
- `internal/app/bobcli/open.go` — loader function rename
- `internal/app/bobcli/app.go` / `init.go` / `usage.go` — add `bob init` command
- `cmd/bob/main_test.go` — env var test patterns remain valid
- `README.md` — setup instructions update
- No new external dependencies (uses JSON, already used by tunnel state)
