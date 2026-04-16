package main

import (
	"io"
	"os"

	"bob/internal/app/bobcli"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return bobcli.Run(args, stdout, stderr)
}
