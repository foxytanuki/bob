package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bob/internal/app/bobcli"
	"bob/internal/version"
)

func TestRunTreatsBareURLAsOpen(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("BOB_SESSION", "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := bobcli.Run([]string{"http://127.0.0.1:8787"}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), "BOB_SESSION is required") {
		t.Fatalf("stderr = %q, want session guidance", stderr.String())
	}
}

func TestRunTreatsBarePortAsLocalhostOpen(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("BOB_SESSION", "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := bobcli.Run([]string{"5173"}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), "BOB_SESSION is required") {
		t.Fatalf("stderr = %q, want session guidance", stderr.String())
	}
}

func TestRunOpenTreatsPortAsLocalhostURL(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("BOB_SESSION", "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := bobcli.Run([]string{"open", "5173"}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), "BOB_SESSION is required") {
		t.Fatalf("stderr = %q, want session guidance", stderr.String())
	}
}

func TestRunKeepsUnknownCommandsAsErrors(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := bobcli.Run([]string{"not-a-command"}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr = %q, want unknown command message", stderr.String())
	}
}

func TestRunVersionCommandPrintsDefaultVersion(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	tests := []struct {
		name string
		args []string
	}{
		{name: "subcommand", args: []string{"version"}},
		{name: "flag-version", args: []string{"--version"}},
		{name: "short-version", args: []string{"-v"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := bobcli.Run(tt.args, &stdout, &stderr)

			if exitCode != 0 {
				t.Fatalf("exitCode = %d, want 0", exitCode)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
			want := "bob " + version.Version
			if got := stdout.String(); !strings.Contains(got, want) {
				t.Fatalf("stdout = %q, want %q", got, want)
			}
		})
	}
}

func TestRunVersionCommandIncludesCommitAndDate(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	oldCommit, oldDate := version.Commit, version.Date
	version.Commit = "abc123"
	version.Date = "2026-03-31"
	t.Cleanup(func() {
		version.Commit = oldCommit
		version.Date = oldDate
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := bobcli.Run([]string{"version"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	got := stdout.String()
	for _, want := range []string{"bob " + version.Version, "commit: abc123", "built: 2026-03-31"} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	}
}

func TestRunVersionCommandRejectsExtraArgs(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := bobcli.Run([]string{"version", "extra"}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if got := stderr.String(); !strings.Contains(got, "usage: bob version") {
		t.Fatalf("stderr = %q, want version usage", got)
	}
}

func TestRunInitWritesConfigWithDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := bobcli.Run([]string{"init", "--token", "cli-token", "--session", "devbox"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0\nstderr = %q", exitCode, stderr.String())
	}

	path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "bob", "bob.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	type writtenCLIConfig struct {
		Endpoint string `json:"endpoint"`
		Token    string `json:"token"`
		Session  string `json:"session"`
		Timeout  string `json:"timeout"`
	}
	var cfg writtenCLIConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	if cfg.Endpoint != "http://127.0.0.1:17331" {
		t.Fatalf("Endpoint = %q, want %q", cfg.Endpoint, "http://127.0.0.1:17331")
	}
	if cfg.Token != "cli-token" {
		t.Fatalf("Token = %q, want %q", cfg.Token, "cli-token")
	}
	if cfg.Session != "devbox" {
		t.Fatalf("Session = %q, want %q", cfg.Session, "devbox")
	}
	if cfg.Timeout != "5s" {
		t.Fatalf("Timeout = %q, want %q", cfg.Timeout, "5s")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want %v", info.Mode().Perm(), os.FileMode(0o600))
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunInitWritesAllProvidedValues(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := bobcli.Run([]string{
		"init",
		"--token", "custom-token",
		"--endpoint", "http://127.0.0.1:9999",
		"--session", "dev",
		"--timeout", "10s",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0\nstderr = %q", exitCode, stderr.String())
	}

	path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "bob", "bob.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	type writtenCLIConfig struct {
		Endpoint string `json:"endpoint"`
		Token    string `json:"token"`
		Session  string `json:"session"`
		Timeout  string `json:"timeout"`
	}
	var cfg writtenCLIConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	if cfg.Endpoint != "http://127.0.0.1:9999" {
		t.Fatalf("Endpoint = %q, want %q", cfg.Endpoint, "http://127.0.0.1:9999")
	}
	if cfg.Token != "custom-token" {
		t.Fatalf("Token = %q, want %q", cfg.Token, "custom-token")
	}
	if cfg.Session != "dev" {
		t.Fatalf("Session = %q, want %q", cfg.Session, "dev")
	}
	if cfg.Timeout != "10s" {
		t.Fatalf("Timeout = %q, want %q", cfg.Timeout, "10s")
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunInitRejectsMissingToken(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := bobcli.Run([]string{"init"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if !strings.Contains(strings.ToLower(stderr.String()), "token") {
		t.Fatalf("stderr = %q, want token-related error", stderr.String())
	}
	if !strings.Contains(strings.ToLower(stderr.String()), "required") {
		t.Fatalf("stderr = %q, want token-required error", stderr.String())
	}
	if strings.Contains(strings.ToLower(stderr.String()), "unknown command") {
		t.Fatalf("stderr = %q, expected command-specific validation, not usage error", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func TestRunInitRejectsMissingSession(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := bobcli.Run([]string{"init", "--token", "cli-token"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	got := strings.ToLower(stderr.String())
	if !strings.Contains(got, "session") || !strings.Contains(got, "required") {
		t.Fatalf("stderr = %q, want session-required error", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func TestRunInitRejectsInvalidTimeout(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := bobcli.Run([]string{"init", "--token", "cli-token", "--session", "devbox", "--timeout", "not-a-duration"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	got := strings.ToLower(stderr.String())
	if !strings.Contains(got, "timeout") {
		t.Fatalf("stderr = %q, want timeout-related error", stderr.String())
	}
	if !(strings.Contains(got, "invalid") || strings.Contains(got, "duration")) {
		t.Fatalf("stderr = %q, want invalid duration error", stderr.String())
	}
	if strings.Contains(got, "unknown command") {
		t.Fatalf("stderr = %q, expected timeout parse validation, not usage error", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func TestRunInitRefusesExistingConfigWithoutForce(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "bob", "bob.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"endpoint":"http://127.0.0.1:17331","token":"old-token","session":"old-session","timeout":"5s"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := bobcli.Run([]string{"init", "--token", "new-token", "--session", "new-session"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "old-token") {
		t.Fatalf("config changed unexpectedly: %q", string(data))
	}
	if !strings.Contains(strings.ToLower(stderr.String()), "already exists") {
		t.Fatalf("stderr = %q, want already exists message", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func TestRunInitForceOverwritesExistingConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "bob", "bob.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"endpoint":"http://127.0.0.1:17331","token":"old-token","session":"old-session","timeout":"5s"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := bobcli.Run([]string{"init", "--token", "new-token", "--session", "new-session", "--force"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0\nstderr = %q", exitCode, stderr.String())
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want %v", info.Mode().Perm(), os.FileMode(0o600))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "new-token") {
		t.Fatalf("config token = %q, want new-token", string(data))
	}
	if strings.Contains(string(data), "old-token") {
		t.Fatalf("old token was not replaced: %q", string(data))
	}
}
