package main

import (
	"io"
	"os"

	"bob/internal/app/bobdapp"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return bobdapp.Run(args, stdout, stderr)
}
