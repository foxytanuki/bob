package tunnel

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type fileLock struct {
	file *os.File
}

func (l *fileLock) Unlock() error {
	if l == nil || l.file == nil {
		return nil
	}
	defer l.file.Close()
	if err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN); err != nil {
		return err
	}
	l.file = nil
	return nil
}

func (m *Manager) sessionFileLock(name string, write bool) (*fileLock, error) {
	return m.sessionFileLockContext(context.Background(), name, write)
}

func (m *Manager) sessionFileLockContext(ctx context.Context, name string, write bool) (*fileLock, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	if err := m.ensureDirs(); err != nil {
		return nil, err
	}
	path := m.lockPath(name)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	flags := syscall.LOCK_SH
	if write {
		flags = syscall.LOCK_EX
	}
	for {
		if err := ctx.Err(); err != nil {
			f.Close()
			return nil, err
		}
		err := syscall.Flock(int(f.Fd()), flags|syscall.LOCK_NB)
		if err == nil {
			return &fileLock{file: f}, nil
		}
		if err != syscall.EWOULDBLOCK && err != syscall.EAGAIN {
			f.Close()
			return nil, err
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			f.Close()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func (m *Manager) globalFileLock(write bool) (*fileLock, error) {
	return m.globalFileLockContext(context.Background(), write)
}

func (m *Manager) globalFileLockContext(ctx context.Context, write bool) (*fileLock, error) {
	if err := m.ensureDirs(); err != nil {
		return nil, err
	}
	path := filepath.Join(m.paths.rootDir, "tunnels.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	flags := syscall.LOCK_SH
	if write {
		flags = syscall.LOCK_EX
	}
	for {
		if err := ctx.Err(); err != nil {
			f.Close()
			return nil, err
		}
		err := syscall.Flock(int(f.Fd()), flags|syscall.LOCK_NB)
		if err == nil {
			return &fileLock{file: f}, nil
		}
		if err != syscall.EWOULDBLOCK && err != syscall.EAGAIN {
			f.Close()
			return nil, err
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			f.Close()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func (m *Manager) sessionLock(name string) *sync.Mutex {
	lock, _ := m.locks.LoadOrStore(name, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func (m *Manager) lockSessionContext(ctx context.Context, name string) (func(), error) {
	lock := m.sessionLock(name)
	for {
		if lock.TryLock() {
			return lock.Unlock, nil
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
