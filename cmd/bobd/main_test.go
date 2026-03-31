package main

import (
	"bytes"
	"testing"

	"bob/internal/tunnel"
)

func TestParseServeOptionsTunnelFlags(t *testing.T) {
	var stderr bytes.Buffer
	opts, err := parseServeOptions([]string{"--tunnel-name", "devbox", "--ssh", "user@remote-host", "--remote-bob-port", "19432", "--local-bobd", "127.0.0.1:7331"}, &stderr)
	if err != nil {
		t.Fatalf("parseServeOptions returned error: %v", err)
	}

	if !opts.tunnelEnabled() {
		t.Fatal("tunnelEnabled = false, want true")
	}
	if opts.tunnelName != "devbox" {
		t.Fatalf("tunnelName = %q, want %q", opts.tunnelName, "devbox")
	}
	if opts.sshTarget != "user@remote-host" {
		t.Fatalf("sshTarget = %q, want %q", opts.sshTarget, "user@remote-host")
	}
	if opts.remoteBobPort != 19432 {
		t.Fatalf("remoteBobPort = %d, want 19432", opts.remoteBobPort)
	}
	if opts.localBobdAddr != "127.0.0.1:7331" {
		t.Fatalf("localBobdAddr = %q, want %q", opts.localBobdAddr, "127.0.0.1:7331")
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
			_, err := parseServeOptions(args, &stderr)
			if err == nil {
				t.Fatal("parseServeOptions returned nil error, want failure")
			}
		})
	}
}

func TestParseServeOptionsDefaultsRemotePort(t *testing.T) {
	var stderr bytes.Buffer
	opts, err := parseServeOptions(nil, &stderr)
	if err != nil {
		t.Fatalf("parseServeOptions returned error: %v", err)
	}
	if opts.remoteBobPort != tunnel.DefaultRemoteBobPort {
		t.Fatalf("remoteBobPort = %d, want %d", opts.remoteBobPort, tunnel.DefaultRemoteBobPort)
	}
}

func TestParseServeOptionsRejectsPositionalArgs(t *testing.T) {
	var stderr bytes.Buffer
	_, err := parseServeOptions([]string{"unexpected"}, &stderr)
	if err == nil {
		t.Fatal("parseServeOptions returned nil error, want failure")
	}
}

func TestServeOptionsLocalBobdAddrOr(t *testing.T) {
	if got := (serveOptions{}).localBobdAddrOr("127.0.0.1:7331"); got != "127.0.0.1:7331" {
		t.Fatalf("localBobdAddrOr fallback = %q, want %q", got, "127.0.0.1:7331")
	}
	if got := (serveOptions{localBobdAddr: "127.0.0.1:19432"}).localBobdAddrOr("127.0.0.1:7331"); got != "127.0.0.1:19432" {
		t.Fatalf("localBobdAddrOr explicit = %q, want %q", got, "127.0.0.1:19432")
	}
}
