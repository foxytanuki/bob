package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadCLI(t *testing.T) {
	t.Run("file loading", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("BOB_ENDPOINT", "")
		t.Setenv("BOB_TOKEN", "")
		t.Setenv("BOB_SESSION", "")
		t.Setenv("BOB_TIMEOUT", "")
		t.Setenv("BOB_CODE_SERVER_PORT", "")

		path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "bob", cliConfigFilename)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		cfg := cliFileConfig{
			Endpoint: "http://127.0.0.1:17331",
			Token:    "cli-token",
			Session:  "demo-session",
			Timeout:  "10s",
			CodeServer: cliCodeServerConfig{
				Port: intPtr(65508),
			},
		}
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}

		got, err := LoadCLI()
		if err != nil {
			t.Fatalf("LoadCLI() error = %v", err)
		}
		if got.Endpoint != cfg.Endpoint {
			t.Fatalf("Endpoint = %q, want %q", got.Endpoint, cfg.Endpoint)
		}
		if got.Token != cfg.Token {
			t.Fatalf("Token = %q, want %q", got.Token, cfg.Token)
		}
		if got.Session != cfg.Session {
			t.Fatalf("Session = %q, want %q", got.Session, cfg.Session)
		}
		if got.Timeout != 10*time.Second {
			t.Fatalf("Timeout = %v, want %v", got.Timeout, 10*time.Second)
		}
		if got.CodeServer.Port != 65508 {
			t.Fatalf("CodeServer.Port = %d, want %d", got.CodeServer.Port, 65508)
		}
	})

	t.Run("missing file defaults", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("BOB_ENDPOINT", "")
		t.Setenv("BOB_TOKEN", "")
		t.Setenv("BOB_SESSION", "")
		t.Setenv("BOB_TIMEOUT", "")
		t.Setenv("BOB_CODE_SERVER_PORT", "")

		got, err := LoadCLI()
		if err != nil {
			t.Fatalf("LoadCLI() error = %v", err)
		}
		if got.Endpoint != defaultCLIEndpoint {
			t.Fatalf("Endpoint = %q, want %q", got.Endpoint, defaultCLIEndpoint)
		}
		if got.Token != "" || got.Session != "" {
			t.Fatalf("Token=%q Session=%q, want empty values", got.Token, got.Session)
		}
		if got.Timeout != 5*time.Second {
			t.Fatalf("Timeout = %v, want %v", got.Timeout, 5*time.Second)
		}
		if got.CodeServer.Port != DefaultCodeServerPort {
			t.Fatalf("CodeServer.Port = %d, want %d", got.CodeServer.Port, DefaultCodeServerPort)
		}
	})

	t.Run("env override", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("BOB_ENDPOINT", "http://env:17331")
		t.Setenv("BOB_TOKEN", "env-token")
		t.Setenv("BOB_SESSION", "env-session")
		t.Setenv("BOB_TIMEOUT", "3s")
		t.Setenv("BOB_CODE_SERVER_PORT", "")

		path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "bob", cliConfigFilename)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(`{"endpoint":"http://file:17331","token":"file-token","session":"file-session","timeout":"10s"}`+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		got, err := LoadCLI()
		if err != nil {
			t.Fatalf("LoadCLI() error = %v", err)
		}
		if got.Endpoint != "http://env:17331" {
			t.Fatalf("Endpoint = %q, want env override", got.Endpoint)
		}
		if got.Token != "env-token" {
			t.Fatalf("Token = %q, want env override", got.Token)
		}
		if got.Session != "env-session" {
			t.Fatalf("Session = %q, want env override", got.Session)
		}
		if got.Timeout != 3*time.Second {
			t.Fatalf("Timeout = %v, want env override", got.Timeout)
		}
		if got.CodeServer.Port != DefaultCodeServerPort {
			t.Fatalf("CodeServer.Port = %d, want default", got.CodeServer.Port)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("BOB_ENDPOINT", "")
		t.Setenv("BOB_TOKEN", "")
		t.Setenv("BOB_SESSION", "")
		t.Setenv("BOB_TIMEOUT", "")
		t.Setenv("BOB_CODE_SERVER_PORT", "")

		path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "bob", cliConfigFilename)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{this is not json}\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		if _, err := LoadCLI(); err == nil {
			t.Fatal("LoadCLI() error = nil, want parse error")
		}
	})

	t.Run("invalid duration", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("BOB_ENDPOINT", "")
		t.Setenv("BOB_TOKEN", "")
		t.Setenv("BOB_SESSION", "")
		t.Setenv("BOB_TIMEOUT", "")
		t.Setenv("BOB_CODE_SERVER_PORT", "")

		path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "bob", cliConfigFilename)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{\"timeout\":\"abc\"}\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		_, err := LoadCLI()
		if err == nil {
			t.Fatal("LoadCLI() error = nil, want duration parse error")
		}
		if !strings.Contains(err.Error(), "BOB_TIMEOUT in config file must be a duration") {
			t.Fatalf("error = %q, want config-file duration parse message", err)
		}
	})

	t.Run("loads code-server port for command-specific validation", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("BOB_ENDPOINT", "")
		t.Setenv("BOB_TOKEN", "")
		t.Setenv("BOB_SESSION", "")
		t.Setenv("BOB_TIMEOUT", "")
		t.Setenv("BOB_CODE_SERVER_PORT", "")

		path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "bob", cliConfigFilename)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{\"codeServer\":{\"port\":0}}\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		got, err := LoadCLI()
		if err != nil {
			t.Fatalf("LoadCLI() error = %v", err)
		}
		if got.CodeServer.Port != 0 {
			t.Fatalf("CodeServer.Port = %d, want raw invalid value for command-specific validation", got.CodeServer.Port)
		}
	})
}

func intPtr(value int) *int {
	return &value
}
