package bobcli

import (
	"flag"
	"fmt"
	"io"
	"time"

	"bob/internal/config"
)

func runInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("bob init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	endpoint := fs.String("endpoint", "http://127.0.0.1:17331", "Forwarded bobd endpoint")
	token := fs.String("token", "", "Bearer token shared with bobd")
	session := fs.String("session", "", "Tunnel/session name used for auto-mirror")
	timeoutRaw := fs.String("timeout", "5s", "Request timeout")
	force := fs.Bool("force", false, "overwrite existing bob config")
	fs.Usage = func() {
		_, _ = io.WriteString(stderr, "Usage: bob init --token <token> --session <name> [--endpoint http://127.0.0.1:17331] [--timeout 5s] [--force]\n")
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() != 0 {
		fs.Usage()
		fmt.Fprintln(stderr, "bob init does not accept positional arguments")
		return 1
	}
	if *token == "" {
		fmt.Fprintln(stderr, "--token is required")
		return 1
	}
	if *session == "" {
		fmt.Fprintln(stderr, "--session is required")
		return 1
	}
	timeout, err := time.ParseDuration(*timeoutRaw)
	if err != nil {
		fmt.Fprintf(stderr, "invalid --timeout duration: %v\n", err)
		return 1
	}

	path, err := config.WriteCLIConfig(config.CLI{
		Endpoint: *endpoint,
		Token:    *token,
		Session:  *session,
		Timeout:  timeout,
	}, *force)
	if err != nil {
		fmt.Fprintf(stderr, "failed to write config: %v\n", err)
		if !*force {
			fmt.Fprintln(stderr, "Use existing config, delete it first, or run 'bob init --force --token <token> --session <name>' to overwrite it.")
		}
		return 1
	}

	fmt.Fprintf(stdout, "Wrote remote CLI config:\n%s\n", path)
	return 0
}
