package tunnel

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"bob/internal/sshwrap"
)

type fakeRunner struct {
	mu           sync.Mutex
	upErr        error
	downErr      error
	checkErr     error
	forwardErrs  []error
	checkErrs    []error
	cancelErr    error
	upCalls      []sshwrap.UpSpec
	downCalls    []sshwrap.ControlSpec
	checkCalls   []sshwrap.ControlSpec
	forwardCalls []sshwrap.ForwardSpec
	cancelCalls  []sshwrap.ForwardSpec
	upSpec       sshwrap.UpSpec
	downSpec     sshwrap.ControlSpec
	checkSpec    sshwrap.ControlSpec
	forwardSpec  sshwrap.ForwardSpec
	cancelSpec   sshwrap.ForwardSpec
}

func (f *fakeRunner) Up(ctx context.Context, spec sshwrap.UpSpec) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.upSpec = spec
	f.upCalls = append(f.upCalls, spec)
	return f.upErr
}

func (f *fakeRunner) Check(ctx context.Context, spec sshwrap.ControlSpec) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.checkSpec = spec
	f.checkCalls = append(f.checkCalls, spec)
	if len(f.checkErrs) > 0 {
		err := f.checkErrs[0]
		f.checkErrs = f.checkErrs[1:]
		return err
	}
	return f.checkErr
}

func (f *fakeRunner) Down(ctx context.Context, spec sshwrap.ControlSpec) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.downSpec = spec
	f.downCalls = append(f.downCalls, spec)
	return f.downErr
}

func (f *fakeRunner) ForwardLocal(ctx context.Context, spec sshwrap.ForwardSpec) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.forwardSpec = spec
	f.forwardCalls = append(f.forwardCalls, spec)
	if len(f.forwardErrs) == 0 {
		return nil
	}
	err := f.forwardErrs[0]
	f.forwardErrs = f.forwardErrs[1:]
	return err
}

func (f *fakeRunner) CancelLocal(ctx context.Context, spec sshwrap.ForwardSpec) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cancelSpec = spec
	f.cancelCalls = append(f.cancelCalls, spec)
	return f.cancelErr
}

func (f *fakeRunner) upCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.upCalls)
}

func (f *fakeRunner) checkCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.checkCalls)
}

func testManager(t *testing.T, root string, runner sshwrap.Runner) *Manager {
	t.Helper()
	return &Manager{
		runner: runner,
		now:    func() time.Time { return time.Unix(123, 0).UTC() },
		paths: paths{
			rootDir:    filepath.Join(root, "bob"),
			tunnelsDir: filepath.Join(root, "bob", "tunnels"),
			controlDir: filepath.Join(root, "bob", "control"),
		},
	}
}

func TestValidateName(t *testing.T) {
	valid := []string{"a", "a1", "A-._z", "bob_1-2.3"}
	for _, name := range valid {
		t.Run(name, func(t *testing.T) {
			if err := ValidateName(name); err != nil {
				t.Fatalf("ValidateName(%q) error = %v", name, err)
			}
		})
	}

	invalid := []string{"", ".bad", " bad", "bad/thing", "-bad"}
	for _, name := range invalid {
		t.Run(name, func(t *testing.T) {
			if err := ValidateName(name); err == nil {
				t.Fatalf("ValidateName(%q) error = nil, want error", name)
			}
		})
	}
}

func TestParsePort(t *testing.T) {
	tests := []struct {
		input string
		want  int
		ok    bool
	}{
		{input: "1", want: 1, ok: true},
		{input: "65535", want: 65535, ok: true},
		{input: " 8080 ", want: 8080, ok: true},
		{input: "", ok: false},
		{input: "abc", ok: false},
		{input: "8080abc", ok: false},
		{input: "0", ok: false},
		{input: "65536", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParsePort(tt.input)
			if tt.ok {
				if err != nil || got != tt.want {
					t.Fatalf("ParsePort(%q) = %d, %v; want %d, nil", tt.input, got, err, tt.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("ParsePort(%q) error = nil, want error", tt.input)
			}
		})
	}
}

func TestNormalizePorts(t *testing.T) {
	got := normalizePorts([]int{8081, 8080, 8081, 80})
	want := []int{80, 8080, 8081}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizePorts() = %#v, want %#v", got, want)
	}
	if normalizePorts(nil) != nil {
		t.Fatal("normalizePorts(nil) = non-nil, want nil")
	}
}

func TestManagerUpCleansUpWhenStateWriteFails(t *testing.T) {
	root := t.TempDir()
	runner := &fakeRunner{}
	m := testManager(t, root, runner)
	statePath := m.statePath("demo")
	if err := os.MkdirAll(filepath.Dir(statePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Dir(statePath), 0o500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(filepath.Dir(statePath), 0o700)

	_, err := m.Up(context.Background(), UpOptions{Name: "demo", SSHTarget: "bob@host"})
	if err == nil {
		t.Fatal("Up() error = nil, want error")
	}
	if _, statErr := os.Stat(m.statePath("demo")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("state file exists after failed Up: %v", statErr)
	}
	if _, statErr := os.Stat(runner.upSpec.ControlSocket); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("control socket exists after failed Up: %v", statErr)
	}
}

func TestManagerDownPreservesStateOnRealRunnerError(t *testing.T) {
	root := t.TempDir()
	runner := &fakeRunner{downErr: errors.New("ssh exited 255")}
	m := testManager(t, root, runner)
	if err := os.MkdirAll(filepath.Dir(m.statePath("demo")), 0o700); err != nil {
		t.Fatal(err)
	}
	state := State{Name: "demo", SSHTarget: "bob@host", ControlSocket: filepath.Join(root, "sock"), CreatedAt: time.Unix(1, 0).UTC()}
	if err := writeState(m.statePath("demo"), state); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(state.ControlSocket, []byte("socket"), 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := m.Down(context.Background(), "demo")
	if !errors.Is(err, runner.downErr) {
		t.Fatalf("Down() err = %v, want %v", err, runner.downErr)
	}
	if res.State.Name != "" || res.Stopped {
		t.Fatalf("Down() result = %#v, want zero value on error", res)
	}
	if _, statErr := os.Stat(m.statePath("demo")); statErr != nil {
		t.Fatalf("state file missing after failed Down: %v", statErr)
	}
}

func TestManagerDownCleansUpWhenControlSocketMissing(t *testing.T) {
	root := t.TempDir()
	runner := &fakeRunner{}
	m := testManager(t, root, runner)
	if err := os.MkdirAll(filepath.Dir(m.statePath("demo")), 0o700); err != nil {
		t.Fatal(err)
	}
	state := State{Name: "demo", SSHTarget: "bob@host", ControlSocket: filepath.Join(root, "missing.sock"), CreatedAt: time.Unix(1, 0).UTC()}
	if err := writeState(m.statePath("demo"), state); err != nil {
		t.Fatal(err)
	}

	res, err := m.Down(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Down() error = %v", err)
	}
	if res.Stopped {
		t.Fatal("Down() stopped = true, want false for missing control socket")
	}
	if _, statErr := os.Stat(m.statePath("demo")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("state file still exists after stale cleanup: %v", statErr)
	}
}

func TestManagerEnsureMirrorReusesExistingMapping(t *testing.T) {
	root := t.TempDir()
	runner := &fakeRunner{}
	m := testManager(t, root, runner)
	state := State{Name: "demo", SSHTarget: "bob@host", ControlSocket: filepath.Join(root, "sock"), CreatedAt: time.Unix(1, 0).UTC(), Mappings: []Mapping{{RemoteHostClass: HostClassLoopback, RemotePort: 8080, LocalPort: 8080, CreatedAt: time.Unix(2, 0).UTC(), LastUsedAt: time.Unix(2, 0).UTC()}}}
	if err := os.MkdirAll(filepath.Dir(m.statePath("demo")), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeState(m.statePath("demo"), state); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(state.ControlSocket, []byte("socket"), 0o600); err != nil {
		t.Fatal(err)
	}

	mapping, reused, err := m.EnsureMirror(context.Background(), "demo", 8080)
	if err != nil {
		t.Fatal(err)
	}
	if !reused || mapping.LocalPort != 8080 {
		t.Fatalf("EnsureMirror() = %#v, %v", mapping, reused)
	}
	if runner.forwardSpec.LocalPort != 0 {
		t.Fatalf("ForwardLocal called unexpectedly: %#v", runner.forwardSpec)
	}
}

func TestManagerEnsureMirrorFallsBackAfterConflict(t *testing.T) {
	root := t.TempDir()
	runner := &fakeRunner{forwardErrs: []error{errors.New("cannot listen to port"), nil}}
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

	mapping, reused, err := m.EnsureMirror(context.Background(), "demo", 8080)
	if err != nil {
		t.Fatal(err)
	}
	if reused {
		t.Fatal("EnsureMirror() reused = true, want false")
	}
	if mapping.LocalPort != FallbackPortStart {
		t.Fatalf("local port = %d, want %d", mapping.LocalPort, FallbackPortStart)
	}
	if runner.forwardSpec.LocalPort != FallbackPortStart {
		t.Fatalf("ForwardLocal local port = %d", runner.forwardSpec.LocalPort)
	}
}

func TestManagerUpCleansUpStaleStateAndSucceeds(t *testing.T) {
	root := t.TempDir()
	runner := &fakeRunner{checkErr: errors.New("control socket missing")}
	m := testManager(t, root, runner)
	state := State{Name: "demo", SSHTarget: "bob@old", ControlSocket: filepath.Join(root, "old.sock"), CreatedAt: time.Unix(1, 0).UTC()}
	if err := os.MkdirAll(filepath.Dir(m.statePath("demo")), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeState(m.statePath("demo"), state); err != nil {
		t.Fatal(err)
	}

	got, err := m.Up(context.Background(), UpOptions{Name: "demo", SSHTarget: "bob@host", MirrorPorts: []int{8080}})
	if err != nil {
		t.Fatal(err)
	}
	if got.SSHTarget != "bob@host" {
		t.Fatalf("Up() target = %q", got.SSHTarget)
	}
	if len(runner.checkCalls) != 1 {
		t.Fatalf("Check calls = %d, want 1", len(runner.checkCalls))
	}
	if len(runner.upCalls) != 1 {
		t.Fatalf("Up calls = %d, want 1", len(runner.upCalls))
	}
	if _, statErr := os.Stat(state.ControlSocket); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("stale socket still exists: %v", statErr)
	}
}

func TestManagerUpPreservesStateOnNonStaleCheckError(t *testing.T) {
	root := t.TempDir()
	runner := &fakeRunner{checkErr: context.DeadlineExceeded}
	m := testManager(t, root, runner)
	state := State{Name: "demo", SSHTarget: "bob@old", ControlSocket: filepath.Join(root, "sock"), CreatedAt: time.Unix(1, 0).UTC()}
	if err := os.MkdirAll(filepath.Dir(m.statePath("demo")), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeState(m.statePath("demo"), state); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(state.ControlSocket, []byte("socket"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := m.Up(context.Background(), UpOptions{Name: "demo", SSHTarget: "bob@host"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Up() err = %v, want %v", err, context.DeadlineExceeded)
	}
	if _, err := m.load("demo"); err != nil {
		t.Fatalf("state should remain: %v", err)
	}
	if len(runner.upCalls) != 0 {
		t.Fatalf("unexpected Up calls: %#v", runner.upCalls)
	}
}

func TestManagerEnsureMirrorSelfHealsStaleSessionAndReusesMapping(t *testing.T) {
	root := t.TempDir()
	runner := &fakeRunner{}
	m := testManager(t, root, runner)
	state := State{Name: "demo", SSHTarget: "bob@host", ControlSocket: filepath.Join(root, "sock"), RemoteBobPort: 17331, LocalBobdAddr: "127.0.0.1:7331", CreatedAt: time.Unix(1, 0).UTC(), Mappings: []Mapping{{RemoteHostClass: HostClassLoopback, RemotePort: 8080, LocalPort: 8080, CreatedAt: time.Unix(2, 0).UTC(), LastUsedAt: time.Unix(2, 0).UTC()}, {RemoteHostClass: HostClassLoopback, RemotePort: 9090, LocalPort: 43210, CreatedAt: time.Unix(3, 0).UTC(), LastUsedAt: time.Unix(3, 0).UTC()}}}
	if err := os.MkdirAll(filepath.Dir(m.statePath("demo")), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeState(m.statePath("demo"), state); err != nil {
		t.Fatal(err)
	}
	runner.checkErr = errors.New("control socket missing")

	mapping, reused, err := m.EnsureMirror(context.Background(), "demo", 8080)
	if err != nil {
		t.Fatal(err)
	}
	if !reused || mapping.LocalPort != 8080 {
		t.Fatalf("EnsureMirror() = %#v, %v", mapping, reused)
	}
	if len(runner.upCalls) != 1 {
		t.Fatalf("expected recovery Up once, got %d", len(runner.upCalls))
	}
	if len(runner.forwardCalls) != 1 || runner.forwardCalls[0].LocalPort != 43210 || runner.forwardCalls[0].RemotePort != 9090 {
		t.Fatalf("forward replay = %#v", runner.forwardCalls)
	}
	if runner.upCalls[0].MirrorPorts == nil || len(runner.upCalls[0].MirrorPorts) != 1 || runner.upCalls[0].MirrorPorts[0] != 8080 {
		t.Fatalf("recovery mirrors = %#v", runner.upCalls[0].MirrorPorts)
	}
}

func TestManagerEnsureMirrorDoesNotRecoverOnContextDeadline(t *testing.T) {
	root := t.TempDir()
	runner := &fakeRunner{checkErr: context.DeadlineExceeded}
	m := testManager(t, root, runner)
	state := State{Name: "demo", SSHTarget: "bob@host", ControlSocket: filepath.Join(root, "sock"), RemoteBobPort: 17331, LocalBobdAddr: "127.0.0.1:7331", CreatedAt: time.Unix(1, 0).UTC()}
	if err := os.MkdirAll(filepath.Dir(m.statePath("demo")), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeState(m.statePath("demo"), state); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(state.ControlSocket, []byte("socket"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, _, err := m.EnsureMirror(context.Background(), "demo", 8080); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("EnsureMirror() err = %v, want %v", err, context.DeadlineExceeded)
	}
	if len(runner.upCalls) != 0 {
		t.Fatalf("unexpected recovery Up calls: %#v", runner.upCalls)
	}
	if _, err := m.load("demo"); err != nil {
		t.Fatalf("state should remain: %v", err)
	}
}

func TestManagerEnsureMirrorSelfHealDefaultsMissingStateFields(t *testing.T) {
	root := t.TempDir()
	runner := &fakeRunner{}
	m := testManager(t, root, runner)
	state := State{Name: "demo", SSHTarget: "bob@host", ControlSocket: filepath.Join(root, "sock"), CreatedAt: time.Unix(1, 0).UTC(), Mappings: []Mapping{{RemoteHostClass: HostClassLoopback, RemotePort: 8080, LocalPort: 8080, CreatedAt: time.Unix(2, 0).UTC(), LastUsedAt: time.Unix(2, 0).UTC()}}}
	if err := os.MkdirAll(filepath.Dir(m.statePath("demo")), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeState(m.statePath("demo"), state); err != nil {
		t.Fatal(err)
	}
	runner.checkErr = errors.New("control socket missing")

	if _, _, err := m.EnsureMirror(context.Background(), "demo", 8080); err != nil {
		t.Fatal(err)
	}
	if len(runner.upCalls) != 1 {
		t.Fatalf("expected recovery Up once, got %d", len(runner.upCalls))
	}
	if runner.upCalls[0].RemoteBobPort != DefaultRemoteBobPort {
		t.Fatalf("remote bob port = %d, want %d", runner.upCalls[0].RemoteBobPort, DefaultRemoteBobPort)
	}
	if runner.upCalls[0].LocalBobdAddr != DefaultLocalBobdAddr {
		t.Fatalf("local bobd addr = %q, want %q", runner.upCalls[0].LocalBobdAddr, DefaultLocalBobdAddr)
	}
}

func TestManagerEnsureMirrorReplaysFallbackLocalPortExactly(t *testing.T) {
	root := t.TempDir()
	runner := &fakeRunner{}
	m := testManager(t, root, runner)
	state := State{Name: "demo", SSHTarget: "bob@host", ControlSocket: filepath.Join(root, "sock"), CreatedAt: time.Unix(1, 0).UTC(), Mappings: []Mapping{{RemoteHostClass: HostClassLoopback, RemotePort: 9090, LocalPort: 43210, CreatedAt: time.Unix(3, 0).UTC(), LastUsedAt: time.Unix(3, 0).UTC()}}}
	if err := os.MkdirAll(filepath.Dir(m.statePath("demo")), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeState(m.statePath("demo"), state); err != nil {
		t.Fatal(err)
	}
	runner.checkErr = errors.New("stale")

	if _, _, err := m.EnsureMirror(context.Background(), "demo", 9090); err != nil {
		t.Fatal(err)
	}
	if len(runner.forwardCalls) != 1 {
		t.Fatalf("forward calls = %d", len(runner.forwardCalls))
	}
	if got := runner.forwardCalls[0]; got.LocalPort != 43210 || got.RemotePort != 9090 {
		t.Fatalf("forward replay = %#v", got)
	}
}

func TestManagerEnsureMirrorRollsBackOnRecoveryFailure(t *testing.T) {
	root := t.TempDir()
	runner := &fakeRunner{forwardErrs: []error{errors.New("replay failed")}}
	m := testManager(t, root, runner)
	state := State{Name: "demo", SSHTarget: "bob@host", ControlSocket: filepath.Join(root, "sock"), CreatedAt: time.Unix(1, 0).UTC(), Mappings: []Mapping{{RemoteHostClass: HostClassLoopback, RemotePort: 9090, LocalPort: 43210, CreatedAt: time.Unix(3, 0).UTC(), LastUsedAt: time.Unix(3, 0).UTC()}}}
	if err := os.MkdirAll(filepath.Dir(m.statePath("demo")), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeState(m.statePath("demo"), state); err != nil {
		t.Fatal(err)
	}
	runner.checkErr = errors.New("stale")

	if _, _, err := m.EnsureMirror(context.Background(), "demo", 9090); err == nil {
		t.Fatal("expected error")
	}
	if len(runner.downCalls) != 1 {
		t.Fatalf("Down calls = %d, want 1", len(runner.downCalls))
	}
	if _, err := m.load("demo"); err != nil {
		t.Fatalf("state should remain after failed recovery: %v", err)
	}
}

func TestWriteStateUsesAtomicRename(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "state.json")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	state := State{Name: "demo", SSHTarget: "bob@host", CreatedAt: time.Unix(1, 0).UTC()}
	if err := writeState(path, state); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "old") {
		t.Fatalf("writeState() left old contents: %s", data)
	}
	if _, err := os.Stat(path + ".tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary file still exists: %v", err)
	}
}

func TestSessionFileLockBlocksAcrossManagers(t *testing.T) {
	root := t.TempDir()
	runner := &fakeRunner{}
	m1 := testManager(t, root, runner)
	m2 := testManager(t, root, runner)
	lock1, err := m1.sessionFileLock("demo", true)
	if err != nil {
		t.Fatal(err)
	}

	started := make(chan struct{})
	acquired := make(chan error, 1)
	go func() {
		close(started)
		lock2, err := m2.sessionFileLock("demo", true)
		if err != nil {
			acquired <- err
			return
		}
		acquired <- lock2.Unlock()
	}()
	<-started
	select {
	case err := <-acquired:
		t.Fatalf("lock acquired too early: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if err := lock1.Unlock(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-acquired:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("lock did not unblock")
	}
}
