package version

import (
	"fmt"
	"io"
)

var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

func Write(w io.Writer, name string) {
	_, _ = fmt.Fprintf(w, "%s %s\n", name, Version)
	if Commit != "" {
		_, _ = fmt.Fprintf(w, "commit: %s\n", Commit)
	}
	if Date != "" {
		_, _ = fmt.Fprintf(w, "built: %s\n", Date)
	}
}
