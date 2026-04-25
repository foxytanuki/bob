package tunnel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bob/internal/sshwrap"
)

const (
	DefaultSupervisorCheckInterval  = 15 * time.Second
	DefaultSupervisorRetryInterval  = 5 * time.Second
	DefaultSupervisorCommandTimeout = 30 * time.Second
)

type SupervisorOptions struct {
	Name           string
	SSHTarget      string
	RemoteBobPort  int
	LocalBobdAddr  string
	CheckInterval  time.Duration
	RetryInterval  time.Duration
	CommandTimeout time.Duration
	Logf           func(format string, args ...any)
}

func (m *Manager) Supervise(ctx context.Context, opts SupervisorOptions) error {
	if err := ValidateName(opts.Name); err != nil {
		return err
	}
	if opts.SSHTarget == "" {
		return errors.New("ssh target is required")
	}
	checkInterval := opts.CheckInterval
	if checkInterval <= 0 {
		checkInterval = DefaultSupervisorCheckInterval
	}
	retryInterval := opts.RetryInterval
	if retryInterval <= 0 {
		retryInterval = DefaultSupervisorRetryInterval
	}
	commandTimeout := opts.CommandTimeout
	if commandTimeout <= 0 {
		commandTimeout = DefaultSupervisorCommandTimeout
	}
	logf := opts.Logf
	if logf == nil {
		logf = func(string, ...any) {}
	}

	for {
		next := checkInterval
		if err := m.superviseOnce(ctx, opts, commandTimeout, logf); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			logf("tunnel supervisor session=%s error: %v", opts.Name, err)
			next = retryInterval
		}

		timer := time.NewTimer(next)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return nil
		case <-timer.C:
		}
	}
}

func (m *Manager) superviseOnce(ctx context.Context, opts SupervisorOptions, commandTimeout time.Duration, logf func(string, ...any)) error {
	unlock, err := m.lockSessionContext(ctx, opts.Name)
	if err != nil {
		return err
	}
	defer unlock()

	fl, err := m.sessionFileLockContext(ctx, opts.Name, true)
	if err != nil {
		return err
	}
	defer fl.Unlock()

	state, err := m.load(opts.Name)
	if errors.Is(err, ErrSessionNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	checkCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	err = m.runner.Check(checkCtx, sshwrap.ControlSpec{Target: state.SSHTarget, ControlSocket: state.ControlSocket})
	cancel()
	if err == nil {
		return nil
	}
	if !isStaleCheckError(state.ControlSocket, err) {
		return fmt.Errorf("check failed: %w", err)
	}

	logf("tunnel supervisor recovering stale session=%s: %v", opts.Name, err)
	recoverCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	recovered, err := m.recoverSessionWithTimeout(recoverCtx, state, commandTimeout)
	if err != nil {
		return fmt.Errorf("recover failed: %w", err)
	}
	logf("tunnel supervisor recovered session=%s endpoint=http://127.0.0.1:%d", recovered.Name, recovered.RemoteBobPort)
	return nil
}
