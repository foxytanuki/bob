package bobcli

import (
	"context"
	"fmt"
	"io"

	"bob/internal/client"
	"bob/internal/config"
	"bob/internal/policy"
)

func runOpen(args []string, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: bob open <url>")
		return 1
	}

	return openURL(args[0], stderr)
}

func openURL(rawURL string, stderr io.Writer) int {
	return openURLWithFailureStatuses(rawURL, stderr, nil)
}

func openURLWithFailureStatuses(rawURL string, stderr io.Writer, failureStatuses map[string]bool) int {
	cfg, err := config.LoadCLI()
	if err != nil {
		fmt.Fprintf(stderr, "config error: %v\n", err)
		return 1
	}
	requiresSession := false
	if parsed, err := policy.NormalizeAndValidate(rawURL, false); err == nil {
		requiresSession = policy.IsLoopbackURL(parsed)
	}
	if requiresSession && cfg.Session == "" {
		fmt.Fprintln(stderr, "BOB_SESSION is required; set it to the tunnel name from 'bob tunnel up'.")
		return 1
	}
	cli := client.New(cfg.Endpoint, cfg.Token, cfg.Timeout)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	resp, err := cli.Open(ctx, rawURL, cfg.Session)
	if err == nil && resp != nil && resp.OK {
		return 0
	}

	displayURL := rawURL
	if resp != nil && resp.OpenedURL != "" {
		displayURL = resp.OpenedURL
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
	if !(requiresSession && resp != nil && (resp.Status == "SESSION_REQUIRED" || resp.Status == "SESSION_NOT_FOUND" || resp.Status == "MIRROR_FAILED")) {
		fmt.Fprintln(stderr, "Open this URL on your local machine:")
		fmt.Fprintln(stderr, displayURL)
	} else {
		fmt.Fprintln(stderr, "This loopback URL needs an active bob session mirror. Check BOB_SESSION and 'bob tunnel status'.")
	}

	if err == nil && resp != nil {
		if failureStatuses[resp.Status] {
			return 1
		}
		switch resp.Status {
		case "UNAUTHORIZED", "INVALID_URL", "INVALID_REQUEST", "DENIED", "INTERNAL_ERROR":
			return 1
		}
	}

	return 0
}

func runDoctor(stdout, stderr io.Writer) int {
	cfg, err := config.LoadCLI()
	if err != nil {
		fmt.Fprintf(stderr, "config error: %v\n", err)
		return 1
	}
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
