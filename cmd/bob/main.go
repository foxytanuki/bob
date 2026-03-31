package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"bob/internal/client"
	"bob/internal/config"
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
	case "open":
		return runOpen(args[1:], stderr)
	case "doctor":
		return runDoctor(stdout, stderr)
	case "help", "--help", "-h":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return 1
	}
}

func runOpen(args []string, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: bob open <url>")
		return 1
	}

	rawURL := args[0]
	cfg := config.LoadCLIFromEnv()
	cli := client.New(cfg.Endpoint, cfg.Token, cfg.Timeout)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	resp, err := cli.Open(ctx, rawURL)
	if err == nil && resp != nil && resp.OK {
		return 0
	}

	fmt.Fprintln(stderr, "Could not open local browser automatically.")
	if err != nil {
		fmt.Fprintf(stderr, "Reason: %v\n", err)
	} else if resp != nil {
		if resp.Message != "" {
			fmt.Fprintf(stderr, "Reason: %s", resp.Message)
			if resp.Status != "" {
				fmt.Fprintf(stderr, " (%s)", resp.Status)
			}
			fmt.Fprintln(stderr)
		} else if resp.Status != "" {
			fmt.Fprintf(stderr, "Reason: %s\n", resp.Status)
		}
	}
	fmt.Fprintln(stderr, "Open this URL on your local machine:")
	fmt.Fprintln(stderr, rawURL)

	if err == nil && resp != nil {
		switch resp.Status {
		case "UNAUTHORIZED", "INVALID_URL", "INVALID_REQUEST", "DENIED", "INTERNAL_ERROR":
			return 1
		}
	}

	return 0
}

func runDoctor(stdout, stderr io.Writer) int {
	cfg := config.LoadCLIFromEnv()
	cli := client.New(cfg.Endpoint, cfg.Token, cfg.Timeout)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	health, err := cli.Health(ctx)
	if err != nil {
		fmt.Fprintf(stderr, "bobd is unreachable via %s: %v\n", cfg.Endpoint, err)
		return 1
	}

	version := health.Version
	if version == "" {
		version = "unknown"
	}

	fmt.Fprintf(stdout, "bobd reachable via %s\n", cfg.Endpoint)
	fmt.Fprintf(stdout, "status: %s\n", health.Status)
	fmt.Fprintf(stdout, "version: %s\n", version)
	fmt.Fprintln(stdout, "note: doctor checks daemon reachability only; it does not verify token correctness")
	return 0
}

func printUsage(w io.Writer) {
	_, _ = io.WriteString(w, `bob - remote to local browser open bridge

Usage:
  bob open <url>
  bob doctor

Environment:
  BOB_ENDPOINT  Forwarded bobd endpoint (default: http://127.0.0.1:17331)
  BOB_TOKEN     Bearer token shared with bobd
  BOB_TIMEOUT   Request timeout (default: 5s)
`)
}
