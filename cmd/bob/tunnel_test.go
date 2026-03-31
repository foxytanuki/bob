package main

import (
	"reflect"
	"testing"
)

func TestSplitLeadingNameKeepsTunnelFlagsInOrder(t *testing.T) {
	name, remaining := splitLeadingName([]string{"ws01", "--ssh", "user@remote-host", "--mirror", "8787"})

	if name != "ws01" {
		t.Fatalf("name = %q, want %q", name, "ws01")
	}

	want := []string{"--ssh", "user@remote-host", "--mirror", "8787"}
	if !reflect.DeepEqual(remaining, want) {
		t.Fatalf("remaining = %#v, want %#v", remaining, want)
	}
}
