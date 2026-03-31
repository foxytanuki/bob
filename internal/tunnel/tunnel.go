package tunnel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"bob/internal/sshwrap"
)

const (
	DefaultRemoteBobPort = 17331
	DefaultLocalBobdAddr = "127.0.0.1:7331"
)

var validNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type State struct {
	Name          string    `json:"name"`
	SSHTarget     string    `json:"ssh_target"`
	ControlSocket string    `json:"control_socket"`
	RemoteBobPort int       `json:"remote_bob_port"`
	LocalBobdAddr string    `json:"local_bobd"`
	MirrorPorts   []int     `json:"mirror_ports,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func (s State) Endpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%d", s.RemoteBobPort)
}

type UpOptions struct {
	Name          string
	SSHTarget     string
	RemoteBobPort int
	LocalBobdAddr string
	MirrorPorts   []int
}

type StatusInfo struct {
	State      State
	Alive      bool
	CheckError string
}

type DownResult struct {
	State   State
	Stopped bool
}

type Manager struct {
	runner sshwrap.Runner
	now    func() time.Time
	paths  paths
}

func NewManager(runner sshwrap.Runner) (*Manager, error) {
	if runner == nil {
		return nil, errors.New("ssh runner is required")
	}
	paths, err := resolvePaths()
	if err != nil {
		return nil, err
	}
	return &Manager{
		runner: runner,
		now:    func() time.Time { return time.Now().UTC() },
		paths:  paths,
	}, nil
}

func (m *Manager) Up(ctx context.Context, opts UpOptions) (State, error) {
	if err := ValidateName(opts.Name); err != nil {
		return State{}, err
	}
	if opts.SSHTarget == "" {
		return State{}, errors.New("ssh target is required")
	}
	if opts.RemoteBobPort == 0 {
		opts.RemoteBobPort = DefaultRemoteBobPort
	}
	if opts.LocalBobdAddr == "" {
		opts.LocalBobdAddr = DefaultLocalBobdAddr
	}
	ports := normalizePorts(opts.MirrorPorts)

	if err := m.ensureDirs(); err != nil {
		return State{}, err
	}

	statePath := m.statePath(opts.Name)
	if _, err := os.Stat(statePath); err == nil {
		return State{}, fmt.Errorf("tunnel %q already exists", opts.Name)
	} else if !errors.Is(err, os.ErrNotExist) {
		return State{}, err
	}

	state := State{
		Name:          opts.Name,
		SSHTarget:     opts.SSHTarget,
		ControlSocket: m.controlSocketPath(opts.Name),
		RemoteBobPort: opts.RemoteBobPort,
		LocalBobdAddr: opts.LocalBobdAddr,
		MirrorPorts:   ports,
		CreatedAt:     m.now(),
	}

	if err := m.runner.Up(ctx, sshwrap.UpSpec{
		Target:        state.SSHTarget,
		ControlSocket: state.ControlSocket,
		RemoteBobPort: state.RemoteBobPort,
		LocalBobdAddr: state.LocalBobdAddr,
		MirrorPorts:   state.MirrorPorts,
	}); err != nil {
		return State{}, err
	}

	if err := writeState(statePath, state); err != nil {
		if cleanupErr := m.cleanupFailedUp(state); cleanupErr != nil {
			return State{}, fmt.Errorf("%w; cleanup also failed: %v", err, cleanupErr)
		}
		return State{}, err
	}

	return state, nil
}

func (m *Manager) Status(ctx context.Context, name string) (StatusInfo, error) {
	state, err := m.load(name)
	if err != nil {
		return StatusInfo{}, err
	}
	return m.check(ctx, state), nil
}

func (m *Manager) StatusAll(ctx context.Context) ([]StatusInfo, error) {
	states, err := m.list()
	if err != nil {
		return nil, err
	}
	statuses := make([]StatusInfo, 0, len(states))
	for _, state := range states {
		statuses = append(statuses, m.check(ctx, state))
	}
	return statuses, nil
}

func (m *Manager) Down(ctx context.Context, name string) (DownResult, error) {
	state, err := m.load(name)
	if err != nil {
		return DownResult{}, err
	}

	if _, err := os.Stat(state.ControlSocket); errors.Is(err, os.ErrNotExist) {
		if err := m.cleanupMetadata(state); err != nil {
			return DownResult{}, err
		}
		return DownResult{State: state, Stopped: false}, nil
	} else if err != nil {
		return DownResult{}, err
	}

	stopped := true
	if err := m.runner.Down(ctx, sshwrap.ControlSpec{Target: state.SSHTarget, ControlSocket: state.ControlSocket}); err != nil {
		if _, statErr := os.Stat(state.ControlSocket); errors.Is(statErr, os.ErrNotExist) {
			if cleanupErr := m.cleanupMetadata(state); cleanupErr != nil {
				return DownResult{}, cleanupErr
			}
			return DownResult{State: state, Stopped: false}, nil
		}
		return DownResult{}, err
	}

	if err := m.cleanupMetadata(state); err != nil {
		return DownResult{}, err
	}

	return DownResult{State: state, Stopped: stopped}, nil
}

func (m *Manager) check(ctx context.Context, state State) StatusInfo {
	err := m.runner.Check(ctx, sshwrap.ControlSpec{Target: state.SSHTarget, ControlSocket: state.ControlSocket})
	if err != nil {
		return StatusInfo{State: state, Alive: false, CheckError: err.Error()}
	}
	return StatusInfo{State: state, Alive: true}
}

func (m *Manager) load(name string) (State, error) {
	if err := ValidateName(name); err != nil {
		return State{}, err
	}
	path := m.statePath(name)
	data, err := os.ReadFile(path)
	if err != nil {
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	state.MirrorPorts = normalizePorts(state.MirrorPorts)
	return state, nil
}

func (m *Manager) list() ([]State, error) {
	if err := m.ensureDirs(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(m.paths.tunnelsDir)
	if err != nil {
		return nil, err
	}
	states := make([]State, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(m.paths.tunnelsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var state State
		if err := json.Unmarshal(data, &state); err != nil {
			return nil, err
		}
		state.MirrorPorts = normalizePorts(state.MirrorPorts)
		states = append(states, state)
	}
	sort.Slice(states, func(i, j int) bool {
		return states[i].Name < states[j].Name
	})
	return states, nil
}

func (m *Manager) ensureDirs() error {
	for _, dir := range []string{m.paths.rootDir, m.paths.tunnelsDir, m.paths.controlDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) statePath(name string) string {
	return filepath.Join(m.paths.tunnelsDir, name+".json")
}

func (m *Manager) controlSocketPath(name string) string {
	prefix := name
	if len(prefix) > 20 {
		prefix = prefix[:20]
	}
	sum := sha256.Sum256([]byte(name))
	return filepath.Join(m.paths.controlDir, fmt.Sprintf("%s-%s.sock", prefix, hex.EncodeToString(sum[:])[:12]))
}

func (m *Manager) cleanupFailedUp(state State) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var errs []error
	if err := m.runner.Down(ctx, sshwrap.ControlSpec{Target: state.SSHTarget, ControlSocket: state.ControlSocket}); err != nil {
		errs = append(errs, err)
	}
	if err := m.cleanupMetadata(state); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (m *Manager) cleanupMetadata(state State) error {
	if err := os.Remove(m.statePath(state.Name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := m.removeSocket(state.ControlSocket); err != nil {
		return err
	}
	return nil
}

func (m *Manager) removeSocket(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

type paths struct {
	rootDir    string
	tunnelsDir string
	controlDir string
}

func resolvePaths() (paths, error) {
	root, err := stateHome()
	if err != nil {
		return paths{}, err
	}
	bobRoot := filepath.Join(root, "bob")
	return paths{
		rootDir:    bobRoot,
		tunnelsDir: filepath.Join(bobRoot, "tunnels"),
		controlDir: filepath.Join(bobRoot, "control"),
	}, nil
}

func stateHome() (string, error) {
	if value := os.Getenv("XDG_STATE_HOME"); value != "" {
		return value, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state"), nil
}

func ValidateName(name string) error {
	if !validNamePattern.MatchString(name) {
		return errors.New("tunnel name must match [A-Za-z0-9][A-Za-z0-9._-]*")
	}
	return nil
}

func ParsePort(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("port is required")
	}
	port, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New("port must be an integer")
	}
	if port < 1 || port > 65535 {
		return 0, errors.New("port must be between 1 and 65535")
	}
	return port, nil
}

func normalizePorts(ports []int) []int {
	if len(ports) == 0 {
		return nil
	}
	unique := make(map[int]struct{}, len(ports))
	for _, port := range ports {
		unique[port] = struct{}{}
	}
	out := make([]int, 0, len(unique))
	for port := range unique {
		out = append(out, port)
	}
	sort.Ints(out)
	return out
}

func writeState(path string, state State) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}
