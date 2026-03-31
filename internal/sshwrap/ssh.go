package sshwrap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

type UpSpec struct {
	Target        string
	ControlSocket string
	RemoteBobPort int
	LocalBobdAddr string
	MirrorPorts   []int
}

type ControlSpec struct {
	Target        string
	ControlSocket string
}

type ForwardSpec struct {
	Target        string
	ControlSocket string
	LocalPort     int
	RemotePort    int
}

type Runner interface {
	Up(ctx context.Context, spec UpSpec) error
	Check(ctx context.Context, spec ControlSpec) error
	Down(ctx context.Context, spec ControlSpec) error
	ForwardLocal(ctx context.Context, spec ForwardSpec) error
	CancelLocal(ctx context.Context, spec ForwardSpec) error
}

type OpenSSH struct {
	binary string
}

func NewOpenSSH() (*OpenSSH, error) {
	path, err := exec.LookPath("ssh")
	if err != nil {
		return nil, err
	}
	return &OpenSSH{binary: path}, nil
}

func (o *OpenSSH) Up(ctx context.Context, spec UpSpec) error {
	args, err := BuildUpArgs(spec)
	if err != nil {
		return err
	}
	return o.run(ctx, args)
}

func (o *OpenSSH) Check(ctx context.Context, spec ControlSpec) error {
	args, err := BuildCheckArgs(spec)
	if err != nil {
		return err
	}
	return o.run(ctx, args)
}

func (o *OpenSSH) Down(ctx context.Context, spec ControlSpec) error {
	args, err := BuildDownArgs(spec)
	if err != nil {
		return err
	}
	return o.run(ctx, args)
}

func (o *OpenSSH) ForwardLocal(ctx context.Context, spec ForwardSpec) error {
	args, err := BuildForwardLocalArgs(spec)
	if err != nil {
		return err
	}
	return o.run(ctx, args)
}

func (o *OpenSSH) CancelLocal(ctx context.Context, spec ForwardSpec) error {
	args, err := BuildCancelLocalArgs(spec)
	if err != nil {
		return err
	}
	return o.run(ctx, args)
}

func (o *OpenSSH) run(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, o.binary, args...)
	var stderr bytes.Buffer
	cmd.Stdout = nil
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return fmt.Errorf("ssh failed: %s", message)
		}
		return err
	}
	return nil
}

func BuildUpArgs(spec UpSpec) ([]string, error) {
	if spec.Target == "" {
		return nil, errors.New("ssh target is required")
	}
	if spec.ControlSocket == "" {
		return nil, errors.New("control socket is required")
	}
	if _, err := splitAndValidateLoopbackAddr(spec.LocalBobdAddr); err != nil {
		return nil, fmt.Errorf("invalid local bobd address: %w", err)
	}
	if err := validatePort(spec.RemoteBobPort); err != nil {
		return nil, fmt.Errorf("invalid remote bob port: %w", err)
	}

	ports := append([]int(nil), spec.MirrorPorts...)
	sort.Ints(ports)
	for _, port := range ports {
		if err := validatePort(port); err != nil {
			return nil, fmt.Errorf("invalid mirror port %d: %w", port, err)
		}
	}

	args := []string{
		"-M",
		"-S", spec.ControlSocket,
		"-fNT",
		"-o", "ExitOnForwardFailure=yes",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=3",
		"-R", fmt.Sprintf("127.0.0.1:%d:%s", spec.RemoteBobPort, spec.LocalBobdAddr),
	}

	for _, port := range ports {
		args = append(args, "-L", fmt.Sprintf("127.0.0.1:%d:127.0.0.1:%d", port, port))
	}

	args = append(args, spec.Target)
	return args, nil
}

func BuildCheckArgs(spec ControlSpec) ([]string, error) {
	if spec.Target == "" {
		return nil, errors.New("ssh target is required")
	}
	if spec.ControlSocket == "" {
		return nil, errors.New("control socket is required")
	}
	return []string{"-S", spec.ControlSocket, "-O", "check", spec.Target}, nil
}

func BuildDownArgs(spec ControlSpec) ([]string, error) {
	if spec.Target == "" {
		return nil, errors.New("ssh target is required")
	}
	if spec.ControlSocket == "" {
		return nil, errors.New("control socket is required")
	}
	return []string{"-S", spec.ControlSocket, "-O", "exit", spec.Target}, nil
}

func BuildForwardLocalArgs(spec ForwardSpec) ([]string, error) {
	if spec.Target == "" {
		return nil, errors.New("ssh target is required")
	}
	if spec.ControlSocket == "" {
		return nil, errors.New("control socket is required")
	}
	if err := validatePort(spec.LocalPort); err != nil {
		return nil, fmt.Errorf("invalid local port: %w", err)
	}
	if err := validatePort(spec.RemotePort); err != nil {
		return nil, fmt.Errorf("invalid remote port: %w", err)
	}
	forward := fmt.Sprintf("127.0.0.1:%d:127.0.0.1:%d", spec.LocalPort, spec.RemotePort)
	return []string{"-S", spec.ControlSocket, "-O", "forward", "-L", forward, spec.Target}, nil
}

func BuildCancelLocalArgs(spec ForwardSpec) ([]string, error) {
	if spec.Target == "" {
		return nil, errors.New("ssh target is required")
	}
	if spec.ControlSocket == "" {
		return nil, errors.New("control socket is required")
	}
	if err := validatePort(spec.LocalPort); err != nil {
		return nil, fmt.Errorf("invalid local port: %w", err)
	}
	if err := validatePort(spec.RemotePort); err != nil {
		return nil, fmt.Errorf("invalid remote port: %w", err)
	}
	forward := fmt.Sprintf("127.0.0.1:%d:127.0.0.1:%d", spec.LocalPort, spec.RemotePort)
	return []string{"-S", spec.ControlSocket, "-O", "cancel", "-L", forward, spec.Target}, nil
}

func splitAndValidateLoopbackAddr(addr string) (int, error) {
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return 0, err
	}
	if host != "127.0.0.1" && host != "localhost" {
		return 0, errors.New("address must use a loopback host")
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return 0, err
	}
	return port, validatePort(port)
}

func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}
	return nil
}
