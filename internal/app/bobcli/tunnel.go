package bobcli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"bob/internal/sshwrap"
	"bob/internal/tunnel"
)

const sshCommandTimeout = 30 * time.Second

func RunTunnel(args []string, stdout, stderr io.Writer) int {
	return runTunnel(args, stdout, stderr)
}

func SplitLeadingName(args []string) (string, []string) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "", args
	}
	return args[0], args[1:]
}

func runTunnel(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printTunnelUsage(stderr)
		return 1
	}

	switch args[0] {
	case "up":
		return runTunnelUp(args[1:], stdout, stderr)
	case "status":
		return runTunnelStatus(args[1:], stdout, stderr)
	case "down":
		return runTunnelDown(args[1:], stdout, stderr)
	case "help", "--help", "-h":
		printTunnelUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown tunnel subcommand: %s\n\n", args[0])
		printTunnelUsage(stderr)
		return 1
	}
}

func runTunnelUp(args []string, stdout, stderr io.Writer) int {
	name, remainingArgs := SplitLeadingName(args)
	fs := flag.NewFlagSet("bob tunnel up", flag.ContinueOnError)
	fs.SetOutput(stderr)
	sshTarget := fs.String("ssh", "", "SSH target, e.g. user@remote-host")
	remoteBobPort := fs.Int("remote-bob-port", tunnel.DefaultRemoteBobPort, "Remote loopback port for bobd control endpoint")
	localBobd := fs.String("local-bobd", tunnel.DefaultLocalBobdAddr, "Local bobd address")
	var mirrors portListFlag
	fs.Var(&mirrors, "mirror", "Mirror remote app port locally (repeatable)")
	fs.Usage = func() {
		_, _ = io.WriteString(stderr, "Usage: bob tunnel up <name> --ssh <target> [--mirror <port>]... [--remote-bob-port 17331] [--local-bobd 127.0.0.1:7331]\n")
	}

	if err := fs.Parse(remainingArgs); err != nil {
		return 1
	}
	if name == "" {
		if fs.NArg() == 1 {
			name = fs.Arg(0)
		} else {
			fs.Usage()
			return 1
		}
	} else if fs.NArg() != 0 {
		fs.Usage()
		return 1
	}
	if *sshTarget == "" {
		fmt.Fprintln(stderr, "--ssh is required")
		return 1
	}

	runner, err := sshwrap.NewOpenSSH()
	if err != nil {
		fmt.Fprintf(stderr, "ssh unavailable: %v\n", err)
		return 1
	}
	mgr, err := tunnel.NewManager(runner)
	if err != nil {
		fmt.Fprintf(stderr, "tunnel manager error: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), sshCommandTimeout)
	defer cancel()

	state, err := mgr.Up(ctx, tunnel.UpOptions{
		Name:          name,
		SSHTarget:     *sshTarget,
		RemoteBobPort: *remoteBobPort,
		LocalBobdAddr: *localBobd,
		MirrorPorts:   mirrors,
	})
	if err != nil {
		fmt.Fprintf(stderr, "failed to start tunnel: %v\n", err)
		return 1
	}

	printTunnelDetails(stdout, tunnel.StatusInfo{State: state, Alive: true})
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "On the remote machine:")
	fmt.Fprintf(stdout, "  export BOB_ENDPOINT=%s\n", state.Endpoint())
	fmt.Fprintln(stdout, "  export BOB_TOKEN=...")
	fmt.Fprintf(stdout, "  export BOB_SESSION=%s\n", state.Name)
	fmt.Fprintln(stdout, "  bob doctor")
	return 0
}

func runTunnelStatus(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("bob tunnel status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	all := fs.Bool("all", false, "Show all tunnels")
	fs.Usage = func() {
		_, _ = io.WriteString(stderr, "Usage: bob tunnel status [<name>|--all]\n")
	}

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() > 1 {
		fs.Usage()
		return 1
	}

	runner, err := sshwrap.NewOpenSSH()
	if err != nil {
		fmt.Fprintf(stderr, "ssh unavailable: %v\n", err)
		return 1
	}
	mgr, err := tunnel.NewManager(runner)
	if err != nil {
		fmt.Fprintf(stderr, "tunnel manager error: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), sshCommandTimeout)
	defer cancel()

	var statuses []tunnel.StatusInfo
	if *all || fs.NArg() == 0 {
		statuses, err = mgr.StatusAll(ctx)
	} else {
		status, statusErr := mgr.Status(ctx, fs.Arg(0))
		if statusErr != nil {
			fmt.Fprintf(stderr, "failed to get tunnel status: %v\n", statusErr)
			return 1
		}
		statuses = []tunnel.StatusInfo{status}
	}
	if err != nil {
		fmt.Fprintf(stderr, "failed to get tunnel status: %v\n", err)
		return 1
	}
	if len(statuses) == 0 {
		fmt.Fprintln(stdout, "No tunnels found.")
		return 0
	}

	anyStale := false
	for i, status := range statuses {
		if i > 0 {
			fmt.Fprintln(stdout)
		}
		printTunnelDetails(stdout, status)
		if !status.Alive {
			anyStale = true
		}
	}

	if anyStale {
		return 1
	}
	return 0
}

func runTunnelDown(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("bob tunnel down", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		_, _ = io.WriteString(stderr, "Usage: bob tunnel down <name>\n")
	}

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 1
	}

	runner, err := sshwrap.NewOpenSSH()
	if err != nil {
		fmt.Fprintf(stderr, "ssh unavailable: %v\n", err)
		return 1
	}
	mgr, err := tunnel.NewManager(runner)
	if err != nil {
		fmt.Fprintf(stderr, "tunnel manager error: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), sshCommandTimeout)
	defer cancel()

	result, err := mgr.Down(ctx, fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "failed to stop tunnel: %v\n", err)
		return 1
	}

	if result.Stopped {
		fmt.Fprintf(stdout, "Tunnel %s stopped.\n", result.State.Name)
	} else {
		fmt.Fprintf(stdout, "Tunnel %s was already stale; local state removed.\n", result.State.Name)
	}
	return 0
}

func printTunnelDetails(w io.Writer, status tunnel.StatusInfo) {
	state := status.State
	stateText := "up"
	if !status.Alive {
		stateText = "stale"
	}

	mirrors := "none"
	if len(state.Mappings) > 0 {
		mirrors = formatMappings(state.Mappings)
	}

	fmt.Fprintf(w, "Tunnel: %s\n", state.Name)
	fmt.Fprintf(w, "SSH target: %s\n", state.SSHTarget)
	fmt.Fprintf(w, "Control endpoint for remote bob: %s\n", state.Endpoint())
	fmt.Fprintf(w, "Mirrors: %s\n", mirrors)
	fmt.Fprintf(w, "Started: %s\n", state.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(w, "Status: %s\n", stateText)
	if status.CheckError != "" {
		fmt.Fprintf(w, "Detail: %s\n", status.CheckError)
	}
}

func formatPorts(ports []int) string {
	parts := make([]string, 0, len(ports))
	for _, port := range ports {
		parts = append(parts, fmt.Sprintf("%d", port))
	}
	return strings.Join(parts, ", ")
}

func formatMappings(mappings []tunnel.Mapping) string {
	parts := make([]string, 0, len(mappings))
	for _, mapping := range mappings {
		parts = append(parts, fmt.Sprintf("%d->%d", mapping.RemotePort, mapping.LocalPort))
	}
	return strings.Join(parts, ", ")
}

type portListFlag []int

func (p *portListFlag) String() string {
	return formatPorts(*p)
}

func (p *portListFlag) Set(value string) error {
	port, err := tunnel.ParsePort(value)
	if err != nil {
		return err
	}
	*p = append(*p, port)
	return nil
}
