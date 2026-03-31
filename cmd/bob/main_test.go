package main

import (
	"bytes"
	"strings"
	"testing"
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
