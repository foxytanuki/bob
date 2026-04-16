package tunnel

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"bob/internal/sshwrap"
)

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
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
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

func (m *Manager) lockPath(name string) string {
	return filepath.Join(m.paths.controlDir, name+".lock")
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

func writeState(path string, state State) error {
	state = normalizeState(state)
	state.MirrorPorts = nil
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
