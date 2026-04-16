package tunnel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"bob/internal/sshwrap"
)

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
	fl, err := m.sessionFileLock(opts.Name, true)
	if err != nil {
		return State{}, err
	}
	defer fl.Unlock()
	gfl, err := m.globalFileLock(true)
	if err != nil {
		return State{}, err
	}
	defer gfl.Unlock()

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
	if existing, err := m.load(opts.Name); err == nil {
		if err := m.runner.Check(ctx, sshwrap.ControlSpec{Target: existing.SSHTarget, ControlSocket: existing.ControlSocket}); err == nil {
			return State{}, fmt.Errorf("tunnel %q already exists", opts.Name)
		} else if !isStaleCheckError(existing.ControlSocket, err) {
			return State{}, err
		}
		if err := m.cleanupMetadata(existing); err != nil {
			return State{}, err
		}
	} else if !errors.Is(err, ErrSessionNotFound) {
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
	fl, err := m.sessionFileLock(name, false)
	if err != nil {
		return StatusInfo{}, err
	}
	defer fl.Unlock()
	state, err := m.load(name)
	if err != nil {
		return StatusInfo{}, err
	}
	return m.check(ctx, state), nil
}

func (m *Manager) StatusAll(ctx context.Context) ([]StatusInfo, error) {
	fl, err := m.globalFileLock(false)
	if err != nil {
		return nil, err
	}
	defer fl.Unlock()
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
	fl, err := m.sessionFileLock(name, true)
	if err != nil {
		return DownResult{}, err
	}
	defer fl.Unlock()
	gfl, err := m.globalFileLock(true)
	if err != nil {
		return DownResult{}, err
	}
	defer gfl.Unlock()

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

func (s State) Endpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%d", s.RemoteBobPort)
}

func (m *Manager) controlSocketPath(name string) string {
	prefix := name
	if len(prefix) > 20 {
		prefix = prefix[:20]
	}
	sum := sha256.Sum256([]byte(name))
	return filepath.Join(m.paths.controlDir, fmt.Sprintf("%s-%s.sock", prefix, hex.EncodeToString(sum[:])[:12]))
}
