## Context

`bob` is a remote→local browser open bridge. It consists of:
- `bob` — remote CLI that sends open requests
- `bobd` — local daemon that receives requests and opens the browser

Currently both tools read all configuration from environment variables. The tunnel subsystem already uses `XDG_STATE_HOME` for persistent state, but there is no equivalent for configuration. Users must set `BOB_TOKEN`, `BOBD_TOKEN`, `BOB_ENDPOINT`, etc. in every shell session.

## Goals / Non-Goals

**Goals:**
- Provide persistent, XDG-compliant configuration storage
- Keep environment variables as an override mechanism
- `bobd init` should write configuration to disk automatically
- `bobd init` should avoid accidental token rotation by refusing to overwrite existing config unless explicitly forced
- `bobd init` should give users a clear path to create remote-side `bob.json`
- `bob init` should create remote-side `bob.json` directly to avoid copy/paste mistakes
- Maintain zero new external dependencies
- Config files with secrets must have restrictive permissions (`0o600`)

**Non-Goals:**
- Keyring / OS credential store integration
- Configuration encryption at rest
- Hot-reload of config without process restart
- Support for multiple profiles / environments within config files

## Decisions

**1. JSON instead of TOML/YAML**
Rationale: The project already uses JSON for tunnel state files and has zero external dependencies. Adding a TOML or YAML parser would be the first dependency. JSON is human-readable enough for the small number of config keys, and `encoding/json` is in the stdlib.
Alternative considered: INI/key-value format. Rejected because it would require writing a custom parser and we already have JSON marshaling code in the project.

**2. Two config files (`bob.json` + `bobd.json`) instead of one**
Rationale: `bob` runs on the remote machine and `bobd` runs on the local machine. They may not share a filesystem. Separate files match the current env var separation (`BOB_*` vs `BOBD_*`) and make it clear which config belongs to which component.
Alternative considered: Single `config.json` with nested `cli` and `daemon` sections. Rejected because it creates a false impression that both tools read the same file.

**3. Env var priority over config file**
Rationale: This is the standard behavior for CLI tools (12-factor app principle). It allows one-off overrides without editing files and keeps containers/CI working.

**4. Config directory: `$XDG_CONFIG_HOME/bob` → `~/.config/bob`**
Rationale: Follows the existing precedent in `internal/tunnel/store.go` which uses `$XDG_STATE_HOME`. Consistent with modern Linux/macOS conventions.

**5. `bobd init` refuses overwrite by default**
Rationale: Re-running `bobd init` can rotate the token and silently break existing remote CLI configuration. The safer default is to fail if `bobd.json` already exists and tell the user how to reset or force regeneration.
Alternative considered: Always overwrite. Rejected because it is surprising and can break active remote setups.

**6. Remote CLI config is both guided by `bobd init` and writable by `bob init`**
Rationale: `bobd init` runs on the local machine, while `bob.json` is usually needed on a remote machine. Writing local `bob.json` would often create config in the wrong place. Instead, `bobd init` prints a ready-to-copy `bob.json` snippet containing the generated token and default endpoint.
To make the remote-side setup less error-prone, `bob init` writes `bob.json` directly on the remote machine from explicit flags.
Alternative considered: Only print snippets from `bobd init`. Rejected because manual JSON creation is easy to mistype and makes config-first setup feel incomplete.

**7. `bob init` requires an explicit token**
Rationale: The token must match the local daemon token. Generating a new remote token would create a broken configuration. `bob init --token <token>` makes the shared-secret relationship explicit.

**8. `bob init` requires an explicit session**
Rationale: The primary use case is opening services hosted on the remote machine, such as `http://127.0.0.1:5173`, in the local browser. Loopback URL mirroring needs a tunnel session name, so requiring `--session` during initialization prevents a later failure during `bob open`.

## Risks / Trade-offs

- **[Risk]** Users may accidentally commit config files containing tokens to version control.
  → **Mitigation**: Default `.gitignore` already ignores `.env`; we will add `~/.config/bob/` is outside typical git repos, but we should document this risk.

- **[Risk]** Config files may be left behind with stale tokens when reinstalling.
  → **Mitigation**: `bobd init` refuses to overwrite by default. Document `rm ~/.config/bob/bobd.json` or `bobd init --force` for intentional reset.

- **[Risk]** Tests that rely on `t.Setenv` may behave differently if a developer has config files on their machine.
  → **Mitigation**: Config tests and any default/fallback tests must set `XDG_CONFIG_HOME=t.TempDir()` so they never read a developer's real config. Env-var override tests still use `t.Setenv`.

## Migration Plan

1. Implement file loaders alongside existing env loaders
2. Update `bobd init` to write config file, refuse existing files by default, support explicit force regeneration, and print optional shell exports plus remote `bob.json` guidance
3. Add `bob init` for writing remote-side `bob.json` with explicit token/session, defaults, and force overwrite behavior
4. Rename `LoadCLIFromEnv` → `LoadCLI`, `LoadDaemonFromEnv` → `LoadDaemon`
5. Update all callers
6. Update README with config-first quick start
7. Run full test suite

Rollback: Delete `~/.config/bob/*.json` to revert to env-only behavior.

## Open Questions

- Should `bob doctor` validate config file presence and permissions?
- Should a future change support reading the token from stdin for shell-safe setup?
