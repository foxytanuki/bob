package tunnel

import (
	"context"
	"fmt"
	"time"

	"bob/internal/sshwrap"
)

const recoverCommandTimeout = 30 * time.Second

func (m *Manager) EnsureMirror(ctx context.Context, session string, remotePort int) (Mapping, bool, error) {
	if err := ValidateName(session); err != nil {
		return Mapping{}, false, err
	}
	lock := m.sessionLock(session)
	lock.Lock()
	defer lock.Unlock()
	fl, err := m.sessionFileLock(session, true)
	if err != nil {
		return Mapping{}, false, err
	}
	defer fl.Unlock()

	if err := validatePort(remotePort); err != nil {
		return Mapping{}, false, err
	}

	state, err := m.load(session)
	if err != nil {
		return Mapping{}, false, err
	}
	if err := m.runner.Check(ctx, sshwrap.ControlSpec{Target: state.SSHTarget, ControlSocket: state.ControlSocket}); err != nil {
		if !isStaleCheckError(state.ControlSocket, err) {
			return Mapping{}, false, fmt.Errorf("failed to verify session %q: %w", session, err)
		}
		if recovered, recErr := m.recoverSession(ctx, state); recErr == nil {
			state = recovered
		} else {
			return Mapping{}, false, fmt.Errorf("session %q is not active: %w (recovery failed: %v)", session, err, recErr)
		}
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
			cleanupErr := m.cancelLocalWithTimeout(spec, recoverCommandTimeout)
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

func (m *Manager) recoverSession(ctx context.Context, state State) (State, error) {
	return m.recoverSessionWithTimeout(ctx, state, recoverCommandTimeout)
}

func (m *Manager) recoverSessionWithTimeout(ctx context.Context, state State, commandTimeout time.Duration) (State, error) {
	if err := m.removeSocket(state.ControlSocket); err != nil {
		return State{}, err
	}
	if state.RemoteBobPort == 0 {
		state.RemoteBobPort = DefaultRemoteBobPort
	}
	if state.LocalBobdAddr == "" {
		state.LocalBobdAddr = DefaultLocalBobdAddr
	}
	mirrorPorts := make([]int, 0, len(state.Mappings))
	for _, mapping := range state.Mappings {
		if mapping.LocalPort == mapping.RemotePort {
			mirrorPorts = append(mirrorPorts, mapping.RemotePort)
		}
	}
	if commandTimeout <= 0 {
		commandTimeout = recoverCommandTimeout
	}
	if err := m.runWithTimeout(ctx, commandTimeout, func(cmdCtx context.Context) error {
		return m.runner.Up(cmdCtx, sshwrap.UpSpec{
			Target:        state.SSHTarget,
			ControlSocket: state.ControlSocket,
			RemoteBobPort: state.RemoteBobPort,
			LocalBobdAddr: state.LocalBobdAddr,
			MirrorPorts:   mirrorPorts,
		})
	}); err != nil {
		_ = m.rollbackRecoveredSession(state, commandTimeout)
		return State{}, err
	}
	for _, mapping := range state.Mappings {
		if mapping.LocalPort == mapping.RemotePort {
			continue
		}
		spec := sshwrap.ForwardSpec{
			Target:        state.SSHTarget,
			ControlSocket: state.ControlSocket,
			LocalPort:     mapping.LocalPort,
			RemotePort:    mapping.RemotePort,
		}
		if err := m.runWithTimeout(ctx, commandTimeout, func(cmdCtx context.Context) error {
			return m.runner.ForwardLocal(cmdCtx, spec)
		}); err != nil {
			if downErr := m.rollbackRecoveredSession(state, commandTimeout); downErr != nil {
				return State{}, fmt.Errorf("%w; rollback failed: %v", err, downErr)
			}
			return State{}, err
		}
	}
	if err := writeState(m.statePath(state.Name), state); err != nil {
		if downErr := m.rollbackRecoveredSession(state, commandTimeout); downErr != nil {
			return State{}, fmt.Errorf("%w; rollback failed: %v", err, downErr)
		}
		return State{}, err
	}
	return state, nil
}

func (m *Manager) runWithTimeout(ctx context.Context, timeout time.Duration, run func(context.Context) error) error {
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return run(cmdCtx)
}

func (m *Manager) rollbackRecoveredSession(state State, timeout time.Duration) error {
	rollbackCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return m.runner.Down(rollbackCtx, sshwrap.ControlSpec{Target: state.SSHTarget, ControlSocket: state.ControlSocket})
}

func (m *Manager) cancelLocalWithTimeout(spec sshwrap.ForwardSpec, timeout time.Duration) error {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return m.runner.CancelLocal(cleanupCtx, spec)
}

func (m *Manager) check(ctx context.Context, state State) StatusInfo {
	err := m.runner.Check(ctx, sshwrap.ControlSpec{Target: state.SSHTarget, ControlSocket: state.ControlSocket})
	if err != nil {
		return StatusInfo{State: state, Alive: false, CheckError: err.Error()}
	}
	return StatusInfo{State: state, Alive: true}
}
