package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"bob/internal/config"
	"bob/internal/opener"
	"bob/internal/server"
	"bob/internal/sshwrap"
	"bob/internal/tunnel"
	"bob/internal/version"
)

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
		return runServe(stderr)
	case "init":
		return runInit(stdout, stderr)
	case "help", "--help", "-h":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return 1
	}
}

func runServe(stderr io.Writer) int {
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
	handler := server.NewHandler(cfg, opener.New(), mgr, logger)
	srv := &http.Server{
		Addr:              cfg.Bind,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

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

func printUsage(w io.Writer) {
	_, _ = io.WriteString(w, `bobd - local daemon for bob

Usage:
  bobd serve
  bobd init

Environment:
  BOBD_BIND            Listen address (default: 127.0.0.1:7331)
  BOBD_TOKEN           Required bearer token
  BOBD_LOCALHOST_ONLY  Allow only loopback URLs (default: true)
`)
}
