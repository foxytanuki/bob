package tunnel

import (
	"errors"
	"regexp"
	"time"
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
