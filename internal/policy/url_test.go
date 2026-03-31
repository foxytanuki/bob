package policy

import "testing"

func TestNormalizeAndValidate_LocalhostOnlyAllowsLoopback(t *testing.T) {
	u, err := NormalizeAndValidate("http://127.0.0.1:8080/path", true)
	if err != nil {
		t.Fatalf("expected loopback URL to be allowed, got error: %v", err)
	}
	if got, want := u.String(), "http://127.0.0.1:8080/path"; got != want {
		t.Fatalf("unexpected normalized URL: got %q want %q", got, want)
	}
}

func TestNormalizeAndValidate_LocalhostOnlyDeniesExternalHost(t *testing.T) {
	_, err := NormalizeAndValidate("https://example.com", true)
	if err != ErrDeniedURL {
		t.Fatalf("expected ErrDeniedURL, got %v", err)
	}
}

func TestNormalizeAndValidate_RejectsNonHTTPST(t *testing.T) {
	for _, raw := range []string{"ftp://example.com", "mailto:test@example.com"} {
		_, err := NormalizeAndValidate(raw, false)
		if err != ErrInvalidURL {
			t.Fatalf("expected ErrInvalidURL for %q, got %v", raw, err)
		}
	}
}

func TestRedactForLogRedactsQueryString(t *testing.T) {
	got := RedactForLog("https://example.com/path?a=1&b=2#frag")
	want := "https://example.com/path?redacted"
	if got != want {
		t.Fatalf("unexpected redacted URL: got %q want %q", got, want)
	}
}
