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
	"sync"
	"time"

	"bob/internal/sshwrap"
)

const (
	DefaultRemoteBobPort = 17331
	DefaultLocalBobdAddr = "127.0.0.1:7331"
	FallbackPortStart    = 43000
	FallbackPortEnd      = 43999
	HostClassLoopback    = "loopback"
)

var (
	validNamePattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	ErrSessionNotFound = errors.New("session not found")
)

type Mapping struct {
	RemoteHostClass string    `json:"remote_host_class"`
	RemotePort      int       `json:"remote_port"`
	LocalPort       int       `json:"local_port"`
	CreatedAt       time.Time `json:"created_at"`
	LastUsedAt      time.Time `json:"last_used_at"`
}

type State struct {
	Name          string    `json:"name"`
	SSHTarget     string    `json:"ssh_target"`
	ControlSocket string    `json:"control_socket"`
	RemoteBobPort int       `json:"remote_bob_port"`
	LocalBobdAddr string    `json:"local_bobd"`
	MirrorPorts   []int     `json:"mirror_ports,omitempty"`
	Mappings      []Mapping `json:"mappings,omitempty"`
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
	locks  sync.Map
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
	lock := m.sessionLock(opts.Name)
	lock.Lock()
	defer lock.Unlock()

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

	now := m.now()
	state := State{
		Name:          opts.Name,
		SSHTarget:     opts.SSHTarget,
		ControlSocket: m.controlSocketPath(opts.Name),
		RemoteBobPort: opts.RemoteBobPort,
		LocalBobdAddr: opts.LocalBobdAddr,
		Mappings:      mappingsFromMirrorPorts(ports, now),
		CreatedAt:     now,
	}

	if err := m.runner.Up(ctx, sshwrap.UpSpec{
		Target:        state.SSHTarget,
		ControlSocket: state.ControlSocket,
		RemoteBobPort: state.RemoteBobPort,
		LocalBobdAddr: state.LocalBobdAddr,
		MirrorPorts:   ports,
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
	lock := m.sessionLock(name)
	lock.Lock()
	defer lock.Unlock()

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

	return DownResult{State: state, Stopped: true}, nil
}

func (m *Manager) EnsureMirror(ctx context.Context, session string, remotePort int) (Mapping, bool, error) {
	if err := ValidateName(session); err != nil {
		return Mapping{}, false, err
	}
	lock := m.sessionLock(session)
	lock.Lock()
	defer lock.Unlock()

	if err := validatePort(remotePort); err != nil {
		return Mapping{}, false, err
	}

	state, err := m.load(session)
	if err != nil {
		return Mapping{}, false, err
	}
	if err := m.runner.Check(ctx, sshwrap.ControlSpec{Target: state.SSHTarget, ControlSocket: state.ControlSocket}); err != nil {
		return Mapping{}, false, fmt.Errorf("session %q is not active: %w", session, err)
	}

	now := m.now()
	for i, mapping := range state.Mappings {
		if mapping.RemoteHostClass == HostClassLoopback && mapping.RemotePort == remotePort {
			state.Mappings[i].LastUsedAt = now
			_ = writeState(m.statePath(session), state)
			return state.Mappings[i], true, nil
		}
	}

	usedLocalPorts := make(map[int]struct{}, len(state.Mappings))
	for _, mapping := range state.Mappings {
		usedLocalPorts[mapping.LocalPort] = struct{}{}
	}

	candidates := candidateLocalPorts(remotePort, usedLocalPorts)
	var lastErr error
	for _, localPort := range candidates {
		spec := sshwrap.ForwardSpec{
			Target:        state.SSHTarget,
			ControlSocket: state.ControlSocket,
			LocalPort:     localPort,
			RemotePort:    remotePort,
		}
		if err := m.runner.ForwardLocal(ctx, spec); err != nil {
			if isPortConflictError(err) {
				lastErr = err
				continue
			}
			return Mapping{}, false, err
		}

		mapping := Mapping{
			RemoteHostClass: HostClassLoopback,
			RemotePort:      remotePort,
			LocalPort:       localPort,
			CreatedAt:       now,
			LastUsedAt:      now,
		}
		state.Mappings = append(state.Mappings, mapping)
		state.Mappings = normalizeMappings(state.Mappings)
		if err := writeState(m.statePath(session), state); err != nil {
			cleanupErr := m.runner.CancelLocal(ctx, spec)
			if cleanupErr != nil {
				return Mapping{}, false, fmt.Errorf("%w; cleanup also failed: %v", err, cleanupErr)
			}
			return Mapping{}, false, err
		}
		return mapping, false, nil
	}

	if lastErr != nil {
		return Mapping{}, false, fmt.Errorf("failed to allocate local mirror for remote port %d: %w", remotePort, lastErr)
	}
	return Mapping{}, false, fmt.Errorf("failed to allocate local mirror for remote port %d", remotePort)
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
		if errors.Is(err, os.ErrNotExist) {
			return State{}, ErrSessionNotFound
		}
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	return normalizeState(state), nil
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
		states = append(states, normalizeState(state))
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

func (m *Manager) sessionLock(name string) *sync.Mutex {
	lock, _ := m.locks.LoadOrStore(name, &sync.Mutex{})
	return lock.(*sync.Mutex)
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
	if err := validatePort(port); err != nil {
		return 0, err
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

func normalizeMappings(mappings []Mapping) []Mapping {
	if len(mappings) == 0 {
		return nil
	}
	byRemotePort := make(map[int]Mapping, len(mappings))
	for _, mapping := range mappings {
		if mapping.RemoteHostClass == "" {
			mapping.RemoteHostClass = HostClassLoopback
		}
		byRemotePort[mapping.RemotePort] = mapping
	}
	out := make([]Mapping, 0, len(byRemotePort))
	for _, mapping := range byRemotePort {
		out = append(out, mapping)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].RemotePort < out[j].RemotePort
	})
	return out
}

func normalizeState(state State) State {
	state.MirrorPorts = normalizePorts(state.MirrorPorts)
	state.Mappings = normalizeMappings(state.Mappings)
	if len(state.MirrorPorts) > 0 {
		migrated := mappingsFromMirrorPorts(state.MirrorPorts, state.CreatedAt)
		state.Mappings = normalizeMappings(append(state.Mappings, migrated...))
		state.MirrorPorts = nil
	}
	return state
}

func mappingsFromMirrorPorts(ports []int, now time.Time) []Mapping {
	ports = normalizePorts(ports)
	mappings := make([]Mapping, 0, len(ports))
	for _, port := range ports {
		mappings = append(mappings, Mapping{
			RemoteHostClass: HostClassLoopback,
			RemotePort:      port,
			LocalPort:       port,
			CreatedAt:       now,
			LastUsedAt:      now,
		})
	}
	return mappings
}

func candidateLocalPorts(remotePort int, used map[int]struct{}) []int {
	ports := make([]int, 0, (FallbackPortEnd-FallbackPortStart)+2)
	if _, exists := used[remotePort]; !exists {
		ports = append(ports, remotePort)
	}
	for port := FallbackPortStart; port <= FallbackPortEnd; port++ {
		if port == remotePort {
			continue
		}
		if _, exists := used[port]; exists {
			continue
		}
		ports = append(ports, port)
	}
	return ports
}

func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}
	return nil
}

func isPortConflictError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "address already in use") ||
		strings.Contains(message, "cannot listen to port") ||
		strings.Contains(message, "port forwarding failed")
}

func writeState(path string, state State) error {
	state = normalizeState(state)
	state.MirrorPorts = nil
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}
