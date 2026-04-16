package bobdapp

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

type ServeOptions struct {
	TunnelName    string
	SSHTarget     string
	RemoteBobPort int
	LocalBobdAddr string
}

func ParseServeOptions(args []string, stderr io.Writer) (ServeOptions, error) {
	opts := ServeOptions{RemoteBobPort: tunnel.DefaultRemoteBobPort}
	fs := flag.NewFlagSet("bobd serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&opts.TunnelName, "tunnel-name", "", "Tunnel session name")
	fs.StringVar(&opts.SSHTarget, "ssh", "", "SSH target, e.g. user@remote-host")
	fs.IntVar(&opts.RemoteBobPort, "remote-bob-port", tunnel.DefaultRemoteBobPort, "Remote loopback port for bobd control endpoint")
	fs.StringVar(&opts.LocalBobdAddr, "local-bobd", "", "Local bobd address forwarded over SSH (default: BOBD_BIND)")
	fs.Usage = func() {
		_, _ = io.WriteString(stderr, "Usage: bobd serve [--tunnel-name <name> --ssh <target>] [--remote-bob-port 17331] [--local-bobd 127.0.0.1:7331]\n")
	}

	if err := fs.Parse(args); err != nil {
		return ServeOptions{}, err
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return ServeOptions{}, errors.New("bobd serve does not accept positional arguments")
	}
	if (opts.TunnelName == "") != (opts.SSHTarget == "") {
		return ServeOptions{}, errors.New("--tunnel-name and --ssh must be provided together")
	}
	return opts, nil
}

func (o ServeOptions) TunnelEnabled() bool {
	return o.TunnelName != ""
}

func (o ServeOptions) LocalBobdAddrOr(fallback string) string {
	if o.LocalBobdAddr != "" {
		return o.LocalBobdAddr
	}
	return fallback
}

func runServe(args []string, stderr io.Writer) int {
	opts, err := ParseServeOptions(args, stderr)
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
	if opts.TunnelEnabled() {
		if mgr == nil {
			fmt.Fprintln(stderr, "tunnel error: ssh is unavailable")
			return 1
		}

		upCtx, cancel := context.WithTimeout(context.Background(), sshCommandTimeout)
		defer cancel()

		if _, err := mgr.Up(upCtx, tunnel.UpOptions{
			Name:          opts.TunnelName,
			SSHTarget:     opts.SSHTarget,
			RemoteBobPort: opts.RemoteBobPort,
			LocalBobdAddr: opts.LocalBobdAddrOr(cfg.Bind),
		}); err != nil {
			fmt.Fprintf(stderr, "failed to start tunnel: %v\n", err)
			return 1
		}
		startedTunnel = true
		logger.Printf("started tunnel session=%s ssh=%s endpoint=http://127.0.0.1:%d", opts.TunnelName, opts.SSHTarget, opts.RemoteBobPort)
		logger.Printf("remote setup: export BOB_ENDPOINT=http://127.0.0.1:%d BOB_SESSION=%s", opts.RemoteBobPort, opts.TunnelName)
	}
	defer func() {
		if !startedTunnel || mgr == nil {
			return
		}
		downCtx, cancel := context.WithTimeout(context.Background(), sshCommandTimeout)
		defer cancel()
		if _, err := mgr.Down(downCtx, opts.TunnelName); err != nil {
			logger.Printf("warning: failed to stop tunnel %s: %v", opts.TunnelName, err)
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
