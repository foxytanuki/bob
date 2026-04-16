package bobdapp

import (
	"fmt"
	"io"

	"bob/internal/cliutil"
)

func Run(args []string, stdout, stderr io.Writer) int {
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
		return cliutil.RunVersion(args, stdout, stderr, "bobd")
	case "help", "--help", "-h":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return 1
	}
}
