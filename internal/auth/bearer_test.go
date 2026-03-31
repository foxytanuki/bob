package auth

import "testing"

func TestValidateBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
		want     bool
	}{
		{name: "valid token", header: "Bearer abc123", expected: "abc123", want: true},
		{name: "missing token", header: "", expected: "abc123", want: false},
		{name: "wrong token", header: "Bearer wrong", expected: "abc123", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateBearerToken(tt.header, tt.expected); got != tt.want {
				t.Fatalf("ValidateBearerToken() = %v, want %v", got, tt.want)
			}
		})
	}
}
