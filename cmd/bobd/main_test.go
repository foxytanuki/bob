package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bob/internal/app/bobdapp"
	"bob/internal/tunnel"
	"bob/internal/version"
)

func TestParseServeOptionsTunnelFlags(t *testing.T) {
	var stderr bytes.Buffer
	opts, err := bobdapp.ParseServeOptions([]string{"--tunnel-name", "devbox", "--ssh", "user@remote-host", "--remote-bob-port", "19432", "--local-bobd", "127.0.0.1:7331"}, &stderr)
	if err != nil {
		t.Fatalf("ParseServeOptions returned error: %v", err)
	}

	if !opts.TunnelEnabled() {
		t.Fatal("TunnelEnabled = false, want true")
	}
	if opts.TunnelName != "devbox" {
		t.Fatalf("TunnelName = %q, want %q", opts.TunnelName, "devbox")
	}
	if opts.SSHTarget != "user@remote-host" {
		t.Fatalf("SSHTarget = %q, want %q", opts.SSHTarget, "user@remote-host")
	}
	if opts.RemoteBobPort != 19432 {
		t.Fatalf("RemoteBobPort = %d, want 19432", opts.RemoteBobPort)
	}
	if opts.LocalBobdAddr != "127.0.0.1:7331" {
		t.Fatalf("LocalBobdAddr = %q, want %q", opts.LocalBobdAddr, "127.0.0.1:7331")
	}
}

func TestParseServeOptionsRejectsPartialTunnelFlags(t *testing.T) {
	tests := [][]string{
		{"--tunnel-name", "devbox"},
		{"--ssh", "user@remote-host"},
	}

	for _, args := range tests {
		t.Run(args[0], func(t *testing.T) {
			var stderr bytes.Buffer
			_, err := bobdapp.ParseServeOptions(args, &stderr)
			if err == nil {
				t.Fatal("ParseServeOptions returned nil error, want failure")
			}
		})
	}
}

func TestParseServeOptionsDefaultsRemotePort(t *testing.T) {
	var stderr bytes.Buffer
	opts, err := bobdapp.ParseServeOptions(nil, &stderr)
	if err != nil {
		t.Fatalf("ParseServeOptions returned error: %v", err)
	}
	if opts.RemoteBobPort != tunnel.DefaultRemoteBobPort {
		t.Fatalf("RemoteBobPort = %d, want %d", opts.RemoteBobPort, tunnel.DefaultRemoteBobPort)
	}
}

func TestParseServeOptionsRejectsPositionalArgs(t *testing.T) {
	var stderr bytes.Buffer
	_, err := bobdapp.ParseServeOptions([]string{"unexpected"}, &stderr)
	if err == nil {
		t.Fatal("ParseServeOptions returned nil error, want failure")
	}
}

func TestServeOptionsLocalBobdAddrOr(t *testing.T) {
	if got := (bobdapp.ServeOptions{}).LocalBobdAddrOr("127.0.0.1:7331"); got != "127.0.0.1:7331" {
		t.Fatalf("LocalBobdAddrOr fallback = %q, want %q", got, "127.0.0.1:7331")
	}
	if got := (bobdapp.ServeOptions{LocalBobdAddr: "127.0.0.1:19432"}).LocalBobdAddrOr("127.0.0.1:7331"); got != "127.0.0.1:19432" {
		t.Fatalf("LocalBobdAddrOr explicit = %q, want %q", got, "127.0.0.1:19432")
	}
}

func TestRunVersionCommandPrintsDefaultVersion(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "flag-version", args: []string{"--version"}},
		{name: "short-version", args: []string{"-v"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := bobdapp.Run(tt.args, &stdout, &stderr)

			if exitCode != 0 {
				t.Fatalf("exitCode = %d, want 0", exitCode)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
			want := "bobd " + version.Version
			if got := stdout.String(); !strings.Contains(got, want) {
				t.Fatalf("stdout = %q, want %q", got, want)
			}
		})
	}
}

func TestRunVersionCommandIncludesCommitAndDate(t *testing.T) {
	oldCommit, oldDate := version.Commit, version.Date
	version.Commit = "abc123"
	version.Date = "2026-03-31"
	t.Cleanup(func() {
		version.Commit = oldCommit
		version.Date = oldDate
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := bobdapp.Run([]string{"--version"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	got := stdout.String()
	for _, want := range []string{"bobd " + version.Version, "commit: abc123", "built: 2026-03-31"} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	}
}

func TestRunVersionCommandRejectsExtraArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := bobdapp.Run([]string{"--version", "extra"}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if got := stderr.String(); !strings.Contains(got, "usage: bobd version") {
		t.Fatalf("stderr = %q, want version usage", got)
	}
}

func TestRunInitWritesConfigAndGuidance(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := bobdapp.Run([]string{"init"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0\nstderr = %q", exitCode, stderr.String())
	}

	path := filepath.Join(configRoot, "bob", "bobd.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var cfg struct {
		Bind          string `json:"bind"`
		Token         string `json:"token"`
		LocalhostOnly *bool  `json:"localhost_only"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	if cfg.Bind != "127.0.0.1:7331" {
		t.Fatalf("config bind = %q, want 127.0.0.1:7331", cfg.Bind)
	}
	if cfg.Token == "" || len(cfg.Token) != 64 {
		t.Fatalf("config token = %q, want non-empty token", cfg.Token)
	}
	if cfg.LocalhostOnly == nil || !*cfg.LocalhostOnly {
		t.Fatal("localhost_only = nil/false, want true")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %v, want %v", info.Mode().Perm(), os.FileMode(0o600))
	}

	out := stdout.String()
	if !strings.Contains(out, path) {
		t.Fatalf("stdout = %q, want path %q", out, path)
	}
	if !strings.Contains(out, "On the remote machine, create ~/.config/bob/bob.json:") {
		t.Fatalf("stdout = %q, want remote bob.json guidance", out)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunInitRefusesExistingConfigByDefault(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)

	path := filepath.Join(configRoot, "bob", "bobd.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"bind":"127.0.0.1:7331","token":"existing-token","localhost_only":true}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := bobdapp.Run([]string{"init"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if out := stderr.String(); !strings.Contains(out, "already exists") {
		t.Fatalf("stderr = %q, want already exists", out)
	}
	if out := stderr.String(); !strings.Contains(out, "Use existing config, delete it first, or run 'bobd init --force' to regenerate it.") {
		t.Fatalf("stderr = %q, want existing config guidance", out)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("unexpected error statting existing config: %v", err)
	}
	if got, err := os.ReadFile(path); err != nil {
		t.Fatal(err)
	} else if !strings.Contains(string(got), "existing-token") {
		t.Fatalf("config token changed unexpectedly: %q", string(got))
	}
}

func TestRunInitForceOverwritesConfig(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)

	path := filepath.Join(configRoot, "bob", "bobd.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"bind":"127.0.0.1:7331","token":"existing-token","localhost_only":true}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := bobdapp.Run([]string{"init", "--force"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0\nstderr = %q", exitCode, stderr.String())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"token"`) {
		t.Fatalf("unexpected config file contents: %q", string(data))
	}

	var cfg struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if cfg.Token == "existing-token" {
		t.Fatalf("token was not regenerated: %q", cfg.Token)
	}
	if cfg.Token == "" {
		t.Fatal("token is empty")
	}

	out := stdout.String()
	if !strings.Contains(out, path) {
		t.Fatalf("stdout = %q, want path %q", out, path)
	}
	if !strings.Contains(out, "On the remote machine, create ~/.config/bob/bob.json:") {
		t.Fatalf("stdout = %q, want remote bob.json guidance", out)
	}
}
