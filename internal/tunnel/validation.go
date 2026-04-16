package tunnel

import (
	"context"
	"errors"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

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
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "address already in use") ||
		strings.Contains(message, "cannot listen to port") ||
		strings.Contains(message, "port forwarding failed")
}

func isStaleCheckError(controlSocket string, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if _, statErr := os.Stat(controlSocket); errors.Is(statErr, os.ErrNotExist) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "control socket connect") ||
		strings.Contains(message, "master is not running") ||
		strings.Contains(message, "no such file or directory") ||
		strings.Contains(message, "connection refused")
}
