package sshwrap

import (
	"reflect"
	"testing"
)

func TestBuildUpArgsIncludesControlAndForwardOptions(t *testing.T) {
	args, err := BuildUpArgs(UpSpec{
		Target:        "bob@host",
		ControlSocket: "/tmp/bob.sock",
		RemoteBobPort: 17331,
		LocalBobdAddr: "127.0.0.1:7331",
		MirrorPorts:   []int{8081, 8080},
	})
	if err != nil {
		t.Fatalf("BuildUpArgs() error = %v", err)
	}

	want := []string{
		"-M",
		"-S", "/tmp/bob.sock",
		"-fNT",
		"-o", "ExitOnForwardFailure=yes",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=3",
		"-R", "127.0.0.1:17331:127.0.0.1:7331",
		"-L", "127.0.0.1:8080:127.0.0.1:8080",
		"-L", "127.0.0.1:8081:127.0.0.1:8081",
		"bob@host",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("BuildUpArgs() = %#v, want %#v", args, want)
	}
}

func TestBuildUpArgsRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		spec UpSpec
	}{
		{name: "missing target", spec: UpSpec{ControlSocket: "/tmp/bob.sock", RemoteBobPort: 17331, LocalBobdAddr: "127.0.0.1:7331"}},
		{name: "bad remote port", spec: UpSpec{Target: "bob@host", ControlSocket: "/tmp/bob.sock", RemoteBobPort: 0, LocalBobdAddr: "127.0.0.1:7331"}},
		{name: "bad local addr", spec: UpSpec{Target: "bob@host", ControlSocket: "/tmp/bob.sock", RemoteBobPort: 17331, LocalBobdAddr: "10.0.0.1:7331"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := BuildUpArgs(tt.spec); err == nil {
				t.Fatal("BuildUpArgs() error = nil, want error")
			}
		})
	}
}
