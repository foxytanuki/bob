## ADDED Requirements

### Requirement: Config directory resolution
The system SHALL resolve the configuration directory using the XDG Base Directory specification.

#### Scenario: XDG_CONFIG_HOME is set
- **WHEN** the `XDG_CONFIG_HOME` environment variable is set
- **THEN** the system SHALL use `$XDG_CONFIG_HOME/bob` as the configuration directory

#### Scenario: XDG_CONFIG_HOME is not set
- **WHEN** the `XDG_CONFIG_HOME` environment variable is not set
- **THEN** the system SHALL fall back to `~/.config/bob` as the configuration directory

### Requirement: CLI configuration file loading
The system SHALL load `bob` CLI configuration from `bob.json` in the configuration directory.

#### Scenario: Config file exists
- **WHEN** `~/.config/bob/bob.json` exists and contains valid JSON
- **THEN** the system SHALL parse `endpoint`, `token`, `session`, and `timeout` fields

#### Scenario: Config file is missing
- **WHEN** `~/.config/bob/bob.json` does not exist
- **THEN** the system SHALL use hardcoded defaults without error

#### Scenario: Config file has invalid JSON
- **WHEN** `~/.config/bob/bob.json` contains malformed JSON
- **THEN** the system SHALL return an error describing the JSON parse failure

### Requirement: Daemon configuration file loading
The system SHALL load `bobd` daemon configuration from `bobd.json` in the configuration directory.

#### Scenario: Config file exists with valid token
- **WHEN** `~/.config/bob/bobd.json` exists and contains a `token` field
- **THEN** the system SHALL load `bind`, `token`, and `localhost_only` fields

#### Scenario: Config file is missing token
- **WHEN** `~/.config/bob/bobd.json` exists but `token` is empty or missing
- **THEN** the system SHALL return an error requiring the user to run `bobd init`

#### Scenario: Config file has invalid JSON
- **WHEN** `~/.config/bob/bobd.json` contains malformed JSON
- **THEN** the system SHALL return an error describing the JSON parse failure

### Requirement: Environment variable override
The system SHALL allow environment variables to override config file values.

#### Scenario: Env var overrides file value
- **GIVEN** `~/.config/bob/bob.json` sets `endpoint` to `http://127.0.0.1:17331`
- **WHEN** the `BOB_ENDPOINT` environment variable is set to `http://example.com:8080`
- **THEN** the system SHALL use `http://example.com:8080` as the endpoint

#### Scenario: Env var provides value when file is missing
- **GIVEN** `~/.config/bob/bob.json` does not exist
- **WHEN** the `BOB_ENDPOINT` environment variable is set to `http://example.com:8080`
- **THEN** the system SHALL use `http://example.com:8080` as the endpoint

#### Scenario: No override uses file value
- **GIVEN** `~/.config/bob/bob.json` sets `endpoint` to `http://127.0.0.1:17331`
- **WHEN** no `BOB_ENDPOINT` environment variable is set
- **THEN** the system SHALL use `http://127.0.0.1:17331` as the endpoint

### Requirement: Config file permissions
The system SHALL create configuration files with restrictive permissions.

#### Scenario: bobd init creates config file
- **WHEN** `bobd init` generates a new configuration file
- **THEN** the file SHALL be created with `0o600` permissions
- **AND** the parent directory SHALL be created with `0o700` permissions if it does not exist

### Requirement: bobd init writes config file
The system SHALL write the generated token to the daemon configuration file during initialization without accidentally overwriting an existing token.

#### Scenario: Init generates and persists token
- **WHEN** the user runs `bobd init`
- **THEN** the system SHALL generate a random token
- **AND** write it to `~/.config/bob/bobd.json`
- **AND** print the token, config file path, optional shell export commands, and a ready-to-copy remote `bob.json` snippet to stdout

#### Scenario: Init refuses existing config by default
- **GIVEN** `~/.config/bob/bobd.json` already exists
- **WHEN** the user runs `bobd init` without a force option
- **THEN** the system SHALL refuse to overwrite the file
- **AND** print guidance for using the existing config or intentionally regenerating it

#### Scenario: Init force overwrites existing config
- **GIVEN** `~/.config/bob/bobd.json` already exists
- **WHEN** the user runs `bobd init` with the force option
- **THEN** the system SHALL generate a new token
- **AND** overwrite `~/.config/bob/bobd.json` with `0o600` permissions

### Requirement: Tests isolate user config
Automated tests for config file loading SHALL isolate themselves from the developer's real configuration directory.

#### Scenario: Config test runs on machine with existing user config
- **GIVEN** the developer has real config files under `~/.config/bob`
- **WHEN** config package tests run
- **THEN** the tests SHALL set `XDG_CONFIG_HOME` to a temporary directory
- **AND** the tests SHALL NOT read or modify the developer's real config files

### Requirement: bob init writes CLI config file
The system SHALL provide a `bob init` command that writes the remote CLI configuration file on the machine where `bob` runs.

#### Scenario: Init creates CLI config with defaults
- **WHEN** the user runs `bob init --token <token> --session <name>`
- **THEN** the system SHALL write `~/.config/bob/bob.json` with `0o600` permissions
- **AND** set `endpoint` to `http://127.0.0.1:17331`
- **AND** set `token` to the provided token
- **AND** set `session` to the provided session
- **AND** set `timeout` to `5s`

#### Scenario: Init creates CLI config with provided values
- **WHEN** the user runs `bob init --token <token> --endpoint <url> --session <name> --timeout <duration>`
- **THEN** the system SHALL write those values to `~/.config/bob/bob.json`

#### Scenario: Init rejects missing token
- **WHEN** the user runs `bob init` without `--token`
- **THEN** the system SHALL fail with a token-required error

#### Scenario: Init rejects missing session
- **WHEN** the user runs `bob init --token <token>` without `--session`
- **THEN** the system SHALL fail with a session-required error

#### Scenario: Init refuses existing CLI config by default
- **GIVEN** `~/.config/bob/bob.json` already exists
- **WHEN** the user runs `bob init --token <token> --session <name>` without a force option
- **THEN** the system SHALL refuse to overwrite the file

#### Scenario: Init force overwrites existing CLI config
- **GIVEN** `~/.config/bob/bob.json` already exists
- **WHEN** the user runs `bob init --token <token> --session <name> --force`
- **THEN** the system SHALL overwrite `~/.config/bob/bob.json` with `0o600` permissions
