package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bob/internal/config"
	"bob/internal/opener"
	"bob/internal/server"
	"bob/internal/sshwrap"
	"bob/internal/tunnel"
	"bob/internal/version"
)

const sshCommandTimeout = 30 * time.Second

type serveOptions struct {
	tunnelName    string
	sshTarget     string
	remoteBobPort int
	localBobdAddr string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 1
	}

	switch args[0] {
	case "serve":
		return runServe(args[1:], stderr)
	case "init":
		return runInit(stdout, stderr)
	case "version", "--version", "-v":
		return runVersion(args, stdout, stderr, "bobd")
	case "help", "--help", "-h":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return 1
	}
}

func runServe(args []string, stderr io.Writer) int {
	opts, err := parseServeOptions(args, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	cfg, err := config.LoadDaemonFromEnv()
	if err != nil {
		fmt.Fprintf(stderr, "config error: %v\n", err)
		return 1
	}

	logger := log.New(stderr, "bobd: ", log.LstdFlags)
	var mgr *tunnel.Manager
	runner, err := sshwrap.NewOpenSSH()
	if err != nil {
		logger.Printf("warning: auto-mirror unavailable: %v", err)
	} else {
		mgr, err = tunnel.NewManager(runner)
		if err != nil {
			logger.Printf("warning: auto-mirror unavailable: %v", err)
		}
	}

	startedTunnel := false
	if opts.tunnelEnabled() {
		if mgr == nil {
			fmt.Fprintln(stderr, "tunnel error: ssh is unavailable")
			return 1
		}

		upCtx, cancel := context.WithTimeout(context.Background(), sshCommandTimeout)
		defer cancel()

		if _, err := mgr.Up(upCtx, tunnel.UpOptions{
			Name:          opts.tunnelName,
			SSHTarget:     opts.sshTarget,
			RemoteBobPort: opts.remoteBobPort,
			LocalBobdAddr: opts.localBobdAddrOr(cfg.Bind),
		}); err != nil {
			fmt.Fprintf(stderr, "failed to start tunnel: %v\n", err)
			return 1
		}
		startedTunnel = true
		logger.Printf("started tunnel session=%s ssh=%s endpoint=http://127.0.0.1:%d", opts.tunnelName, opts.sshTarget, opts.remoteBobPort)
		logger.Printf("remote setup: export BOB_ENDPOINT=http://127.0.0.1:%d BOB_SESSION=%s", opts.remoteBobPort, opts.tunnelName)
	}
	defer func() {
		if !startedTunnel || mgr == nil {
			return
		}
		downCtx, cancel := context.WithTimeout(context.Background(), sshCommandTimeout)
		defer cancel()
		if _, err := mgr.Down(downCtx, opts.tunnelName); err != nil {
			logger.Printf("warning: failed to stop tunnel %s: %v", opts.tunnelName, err)
		}
	}()

	handler := server.NewHandler(cfg, opener.New(), mgr, logger)
	srv := &http.Server{
		Addr:              cfg.Bind,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Printf("shutdown error: %v", err)
		}
	}()

	logger.Printf("starting bobd %s on http://%s", version.Version, cfg.Bind)
	err = srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Printf("server error: %v", err)
		return 1
	}

	return 0
}

func parseServeOptions(args []string, stderr io.Writer) (serveOptions, error) {
	opts := serveOptions{remoteBobPort: tunnel.DefaultRemoteBobPort}
	fs := flag.NewFlagSet("bobd serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&opts.tunnelName, "tunnel-name", "", "Tunnel session name")
	fs.StringVar(&opts.sshTarget, "ssh", "", "SSH target, e.g. user@remote-host")
	fs.IntVar(&opts.remoteBobPort, "remote-bob-port", tunnel.DefaultRemoteBobPort, "Remote loopback port for bobd control endpoint")
	fs.StringVar(&opts.localBobdAddr, "local-bobd", "", "Local bobd address forwarded over SSH (default: BOBD_BIND)")
	fs.Usage = func() {
		_, _ = io.WriteString(stderr, "Usage: bobd serve [--tunnel-name <name> --ssh <target>] [--remote-bob-port 17331] [--local-bobd 127.0.0.1:7331]\n")
	}

	if err := fs.Parse(args); err != nil {
		return serveOptions{}, err
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return serveOptions{}, errors.New("bobd serve does not accept positional arguments")
	}
	if (opts.tunnelName == "") != (opts.sshTarget == "") {
		return serveOptions{}, errors.New("--tunnel-name and --ssh must be provided together")
	}
	return opts, nil
}

func (o serveOptions) tunnelEnabled() bool {
	return o.tunnelName != ""
}

func (o serveOptions) localBobdAddrOr(fallback string) string {
	if o.localBobdAddr != "" {
		return o.localBobdAddr
	}
	return fallback
}

func runInit(stdout, stderr io.Writer) int {
	token, err := generateToken()
	if err != nil {
		fmt.Fprintf(stderr, "failed to generate token: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "Generated token:\n%s\n\n", token)
	_, _ = fmt.Fprintf(stdout, "Set on the local machine before running bobd:\n")
	_, _ = fmt.Fprintf(stdout, "  export BOBD_TOKEN=%s\n\n", token)
	_, _ = fmt.Fprintf(stdout, "Set on the remote machine for bob:\n")
	_, _ = fmt.Fprintf(stdout, "  export BOB_TOKEN=%s\n", token)
	_, _ = fmt.Fprintf(stdout, "  export BOB_ENDPOINT=http://127.0.0.1:17331\n")
	return 0
}

func generateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func runVersion(args []string, stdout, stderr io.Writer, name string) int {
	if len(args) != 1 {
		fmt.Fprintf(stderr, "usage: %s version\n", name)
		return 1
	}
	version.Write(stdout, name)
	return 0
}

func printUsage(w io.Writer) {
	_, _ = io.WriteString(w, `bobd - local daemon for bob

Usage:
  bobd serve [--tunnel-name <name> --ssh <target>] [--remote-bob-port 17331] [--local-bobd 127.0.0.1:7331]
  bobd init
  bobd version

Environment:
  BOBD_BIND            Listen address (default: 127.0.0.1:7331)
  BOBD_TOKEN           Required bearer token
  BOBD_LOCALHOST_ONLY  Allow only loopback URLs (default: true)
`)
}
