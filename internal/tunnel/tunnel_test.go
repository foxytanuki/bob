package tunnel

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"bob/internal/sshwrap"
)

type fakeRunner struct {
	upErr    error
	downErr  error
	upSpec   sshwrap.UpSpec
	downSpec sshwrap.ControlSpec
}

func (f *fakeRunner) Up(ctx context.Context, spec sshwrap.UpSpec) error {
	f.upSpec = spec
	return f.upErr
}

func (f *fakeRunner) Check(ctx context.Context, spec sshwrap.ControlSpec) error { return nil }

func (f *fakeRunner) Down(ctx context.Context, spec sshwrap.ControlSpec) error {
	f.downSpec = spec
	return f.downErr
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
