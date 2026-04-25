package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDaemon(t *testing.T) {
	t.Run("file loading", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("BOBD_BIND", "")
		t.Setenv("BOBD_TOKEN", "")
		t.Setenv("BOBD_LOCALHOST_ONLY", "")

		path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "bob", daemonConfigFilename)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		cfg := daemonFileConfig{
			Bind:  "127.0.0.1:9000",
			Token: "daemon-token",
			LocalhostOnly: func() *bool {
				v := false
				return &v
			}(),
		}
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}

		got, err := LoadDaemon()
		if err != nil {
			t.Fatalf("LoadDaemon() error = %v", err)
		}
		if got.Bind != cfg.Bind {
			t.Fatalf("Bind = %q, want %q", got.Bind, cfg.Bind)
		}
		if got.Token != cfg.Token {
			t.Fatalf("Token = %q, want %q", got.Token, cfg.Token)
		}
		if got.LocalhostOnly != false {
			t.Fatalf("LocalhostOnly = %v, want false", got.LocalhostOnly)
		}
	})

	t.Run("missing token error", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("BOBD_BIND", "")
		t.Setenv("BOBD_TOKEN", "")
		t.Setenv("BOBD_LOCALHOST_ONLY", "")

		if _, err := LoadDaemon(); err == nil {
			t.Fatal("LoadDaemon() error = nil, want token required error")
		}
	})

	t.Run("env override", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("BOBD_BIND", "0.0.0.0:7331")
		t.Setenv("BOBD_TOKEN", "env-token")
		t.Setenv("BOBD_LOCALHOST_ONLY", "false")

		path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "bob", daemonConfigFilename)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{\"bind\":\"127.0.0.1:9000\",\"token\":\"file-token\",\"localhost_only\":true}\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		got, err := LoadDaemon()
		if err != nil {
			t.Fatalf("LoadDaemon() error = %v", err)
		}
		if got.Bind != "0.0.0.0:7331" {
			t.Fatalf("Bind = %q, want env override", got.Bind)
		}
		if got.Token != "env-token" {
			t.Fatalf("Token = %q, want env override", got.Token)
		}
		if got.LocalhostOnly != false {
			t.Fatalf("LocalhostOnly = %v, want false", got.LocalhostOnly)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("BOBD_TOKEN", "")
		t.Setenv("BOBD_BIND", "")
		t.Setenv("BOBD_LOCALHOST_ONLY", "")

		path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "bob", daemonConfigFilename)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{invalid json}\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		if _, err := LoadDaemon(); err == nil {
			t.Fatal("LoadDaemon() error = nil, want parse error")
		}
	})

	t.Run("invalid BOBD_LOCALHOST_ONLY", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("BOBD_TOKEN", "env-token")
		t.Setenv("BOBD_LOCALHOST_ONLY", "not-a-bool")

		if _, err := LoadDaemon(); err == nil {
			t.Fatal("LoadDaemon() error = nil, want bool parse error")
		} else if !strings.Contains(err.Error(), "BOBD_LOCALHOST_ONLY must be a boolean") {
			t.Fatalf("error = %q, want bool parse message", err)
		}
	})
}

func TestWriteDaemonConfig(t *testing.T) {
	t.Run("refuses existing config unless force", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "bob", daemonConfigFilename)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(`{"bind":"127.0.0.1:7331","token":"old-token","localhost_only":true}`), 0o600); err != nil {
			t.Fatal(err)
		}

		_, err := WriteDaemonConfig(Daemon{Bind: "127.0.0.1:7331", Token: "new-token", LocalhostOnly: true}, false)
		if err == nil {
			t.Fatal("WriteDaemonConfig() error = nil, want refusal")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("error = %q, want already exists", err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(got), "old-token") {
			t.Fatalf("existing config was overwritten: %q", string(got))
		}
	})

	t.Run("force overwrite and mode", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "bob", daemonConfigFilename)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(`{"bind":"127.0.0.1:7331","token":"old-token","localhost_only":true}`), 0o644); err != nil {
			t.Fatal(err)
		}

		written, err := WriteDaemonConfig(Daemon{Bind: "0.0.0.0:7331", Token: "new-token", LocalhostOnly: false}, true)
		if err != nil {
			t.Fatalf("WriteDaemonConfig() error = %v", err)
		}
		if written != path {
			t.Fatalf("path = %q, want %q", written, path)
		}

		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("mode = %v, want %v", info.Mode().Perm(), os.FileMode(0o600))
		}

		var got daemonFileConfig
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
		if got.Bind != "0.0.0.0:7331" || got.Token != "new-token" || got.LocalhostOnly == nil || *got.LocalhostOnly {
			t.Fatalf("config = %#v", got)
		}
	})
}
