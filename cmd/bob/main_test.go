package main

import (
	"bytes"
	"strings"
	"testing"

	"bob/internal/version"
)

func TestRunTreatsBareURLAsOpen(t *testing.T) {
	t.Setenv("BOB_SESSION", "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"http://127.0.0.1:8787"}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), "BOB_SESSION is required") {
		t.Fatalf("stderr = %q, want session guidance", stderr.String())
	}
}

func TestRunKeepsUnknownCommandsAsErrors(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"not-a-command"}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr = %q, want unknown command message", stderr.String())
	}
}

func TestRunVersionCommandPrintsDefaultVersion(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "subcommand", args: []string{"version"}},
		{name: "flag-version", args: []string{"--version"}},
		{name: "short-version", args: []string{"-v"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := run(tt.args, &stdout, &stderr)

			if exitCode != 0 {
				t.Fatalf("exitCode = %d, want 0", exitCode)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
			if got := stdout.String(); !strings.Contains(got, "bob dev") {
				t.Fatalf("stdout = %q, want bob dev", got)
			}
		})
	}
}

func TestRunVersionCommandIncludesCommitAndDate(t *testing.T) {
	oldCommit, oldDate := version.Commit, version.Date
	version.Commit = "abc123"
	version.Date = "2026-03-31"
	t.Cleanup(func() {
		version.Commit = oldCommit
		version.Date = oldDate
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"version"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	got := stdout.String()
	for _, want := range []string{"bob dev", "commit: abc123", "built: 2026-03-31"} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	}
}

func TestRunVersionCommandRejectsExtraArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"version", "extra"}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if got := stderr.String(); !strings.Contains(got, "usage: bob version") {
		t.Fatalf("stderr = %q, want version usage", got)
	}
}
