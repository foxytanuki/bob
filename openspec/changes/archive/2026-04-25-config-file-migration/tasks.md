## 1. Core Config File Infrastructure

- [x] 1.1 Create `internal/config/file.go` with XDG config directory resolution (`$XDG_CONFIG_HOME/bob` → `~/.config/bob`)
- [x] 1.2 Add generic JSON config loader helper in `internal/config/file.go`
- [x] 1.3 Add `ensureConfigDir()` helper that creates directory with `0o700` permissions

## 2. Update CLI Configuration Loading

- [x] 2.1 Rename `LoadCLIFromEnv()` → `LoadCLI()` in `internal/config/cli.go`
- [x] 2.2 Implement config file loading with env var override: load `~/.config/bob/bob.json` first, then apply `BOB_*` env vars
- [x] 2.3 Add tests in `internal/config/cli_test.go` for file loading, env override, missing file, and invalid JSON using `XDG_CONFIG_HOME=t.TempDir()`

## 3. Update Daemon Configuration Loading

- [x] 3.1 Rename `LoadDaemonFromEnv()` → `LoadDaemon()` in `internal/config/daemon.go`
- [x] 3.2 Implement config file loading with env var override: load `~/.config/bob/bobd.json` first, then apply `BOBD_*` env vars
- [x] 3.3 Add tests in `internal/config/daemon_test.go` for file loading, missing token, env override, and invalid JSON using `XDG_CONFIG_HOME=t.TempDir()`

## 4. Update bobd init Command

- [x] 4.1 Update `runInit()` in `internal/app/bobdapp/serve.go` to write generated token to `~/.config/bob/bobd.json` with `0o600` permissions
- [x] 4.2 Make `bobd init` refuse to overwrite existing `bobd.json` by default
- [x] 4.3 Add explicit force option for `bobd init` to regenerate and overwrite existing daemon config
- [x] 4.4 Update `runInit()` output message to confirm config file path, show optional shell exports, and print a ready-to-copy remote `bob.json` snippet
- [x] 4.5 Add tests for default refusal, force overwrite, file permissions, and output guidance

## 5. Update Callers

- [x] 5.1 Update `internal/app/bobcli/open.go` to use `config.LoadCLI()` instead of `config.LoadCLIFromEnv()`
- [x] 5.2 Update `internal/app/bobdapp/serve.go` to use `config.LoadDaemon()` instead of `config.LoadDaemonFromEnv()`
- [x] 5.3 Update test in `cmd/bob/main_test.go` that references `BOB_SESSION` (env var override pattern remains valid)

## 6. Documentation

- [x] 6.1 Update `README.md` quick start to use config-first setup (`bobd init` writes file, no manual exports needed)
- [x] 6.2 Update `README.md` environment section to explain env vars override config file values
- [x] 6.3 Document that remote machines need `~/.config/bob/bob.json`, using the snippet printed by `bobd init`
- [x] 6.4 Document existing-config behavior and force regeneration/reset path
- [x] 6.5 Update `.gitignore` if needed (config is in `~/.config/bob/` so generally outside repo, but verify)

## 7. Verification

- [x] 7.1 Run `go test ./...` and ensure all tests pass
- [x] 7.2 Manual test: `bobd init` creates `~/.config/bob/bobd.json` with correct permissions
- [x] 7.3 Manual test: `bob` reads `~/.config/bob/bob.json` and env vars override file values
- [x] 7.4 Clean up test config files after manual verification

## 8. Add bob init Command

- [x] 8.1 Add config writer for `~/.config/bob/bob.json` with `0o600` permissions and force overwrite behavior
- [x] 8.2 Add `bob init --token <token> --session <name> [--endpoint <url>] [--timeout <duration>] [--force]`
- [x] 8.3 Update `bob` usage text to document `init`
- [x] 8.4 Add tests for defaults, provided values, missing token, missing session, invalid timeout, existing config refusal, and force overwrite
- [x] 8.5 Update README to prefer `bob init` on the remote machine
- [x] 8.6 Run `go test ./...` and verify all tests pass
