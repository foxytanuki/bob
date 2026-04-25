package tunnel

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManagerSupervisorRecoversStaleSession(t *testing.T) {
	root := t.TempDir()
	runner := &fakeRunner{checkErrs: []error{errors.New("control socket missing")}}
	m := testManager(t, root, runner)
	state := State{
		Name:          "demo",
		SSHTarget:     "bob@host",
		ControlSocket: filepath.Join(root, "missing.sock"),
		RemoteBobPort: 17331,
		LocalBobdAddr: "127.0.0.1:7331",
		CreatedAt:     time.Unix(1, 0).UTC(),
		Mappings: []Mapping{
			{RemoteHostClass: HostClassLoopback, RemotePort: 8080, LocalPort: 8080, CreatedAt: time.Unix(2, 0).UTC(), LastUsedAt: time.Unix(2, 0).UTC()},
			{RemoteHostClass: HostClassLoopback, RemotePort: 9090, LocalPort: 43210, CreatedAt: time.Unix(3, 0).UTC(), LastUsedAt: time.Unix(3, 0).UTC()},
		},
	}
	if err := os.MkdirAll(filepath.Dir(m.statePath("demo")), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeState(m.statePath("demo"), state); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- m.Supervise(ctx, SupervisorOptions{
			Name:           "demo",
			SSHTarget:      "bob@host",
			CheckInterval:  time.Hour,
			RetryInterval:  time.Millisecond,
			CommandTimeout: time.Second,
		})
	}()

	waitFor(t, time.Second, func() bool { return runner.upCallCount() == 1 })
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Supervise() error = %v", err)
	}
	if runner.upCalls[0].RemoteBobPort != 17331 {
		t.Fatalf("recovered remote bob port = %d", runner.upCalls[0].RemoteBobPort)
	}
	if len(runner.upCalls[0].MirrorPorts) != 1 || runner.upCalls[0].MirrorPorts[0] != 8080 {
		t.Fatalf("recovered mirror ports = %#v", runner.upCalls[0].MirrorPorts)
	}
	if len(runner.forwardCalls) != 1 || runner.forwardCalls[0].LocalPort != 43210 || runner.forwardCalls[0].RemotePort != 9090 {
		t.Fatalf("replayed forwards = %#v", runner.forwardCalls)
	}
}

func TestManagerSupervisorDoesNotRecoverNonStaleCheckError(t *testing.T) {
	root := t.TempDir()
	runner := &fakeRunner{checkErr: context.DeadlineExceeded}
	m := testManager(t, root, runner)
	state := State{Name: "demo", SSHTarget: "bob@host", ControlSocket: filepath.Join(root, "sock"), CreatedAt: time.Unix(1, 0).UTC()}
	if err := os.MkdirAll(filepath.Dir(m.statePath("demo")), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeState(m.statePath("demo"), state); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(state.ControlSocket, []byte("socket"), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- m.Supervise(ctx, SupervisorOptions{
			Name:           "demo",
			SSHTarget:      "bob@host",
			CheckInterval:  time.Hour,
			RetryInterval:  time.Hour,
			CommandTimeout: time.Second,
		})
	}()

	waitFor(t, time.Second, func() bool { return runner.checkCallCount() == 1 })
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Supervise() error = %v", err)
	}
	if len(runner.upCalls) != 0 {
		t.Fatalf("unexpected recovery Up calls: %#v", runner.upCalls)
	}
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}
