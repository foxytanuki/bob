package tunnel

import (
	"os"
	"path/filepath"
	"sync"
	"syscall"
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
	if err := syscall.Flock(int(f.Fd()), flags); err != nil {
		f.Close()
		return nil, err
	}
	return &fileLock{file: f}, nil
}

func (m *Manager) globalFileLock(write bool) (*fileLock, error) {
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
	if err := syscall.Flock(int(f.Fd()), flags); err != nil {
		f.Close()
		return nil, err
	}
	return &fileLock{file: f}, nil
}

func (m *Manager) sessionLock(name string) *sync.Mutex {
	lock, _ := m.locks.LoadOrStore(name, &sync.Mutex{})
	return lock.(*sync.Mutex)
}
