package cliutil

import (
	"fmt"
	"io"

	"bob/internal/version"
)

func RunVersion(args []string, stdout, stderr io.Writer, name string) int {
	if len(args) != 1 {
		fmt.Fprintf(stderr, "usage: %s version\n", name)
		return 1
	}
	version.Write(stdout, name)
	return 0
}
