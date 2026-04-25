package bobcli

import (
	"fmt"
	"io"

	"bob/internal/cliutil"
	"bob/internal/policy"
)

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 1
	}
	if len(args) == 1 && looksLikeURL(args[0]) {
		return runOpen(args, stderr)
	}

	switch args[0] {
	case "init":
		return runInit(args[1:], stdout, stderr)
	case "open":
		return runOpen(args[1:], stderr)
	case "code-server":
		return runCodeServer(args[1:], stderr)
	case "doctor":
		return runDoctor(stdout, stderr)
	case "version", "--version", "-v":
		return cliutil.RunVersion(args, stdout, stderr, "bob")
	case "tunnel":
		return runTunnel(args[1:], stdout, stderr)
	case "help", "--help", "-h":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return 1
	}
}

func looksLikeURL(value string) bool {
	parsed, err := policy.NormalizeAndValidate(value, false)
	return err == nil && parsed != nil
}
