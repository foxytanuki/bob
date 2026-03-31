package policy

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

var (
	ErrInvalidURL = errors.New("invalid url")
	ErrDeniedURL  = errors.New("url denied by policy")
)

func NormalizeAndValidate(raw string, localhostOnly bool) (*url.URL, error) {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return nil, ErrInvalidURL
	}

	parsed.Scheme = strings.ToLower(parsed.Scheme)
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, ErrInvalidURL
	}

	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "" {
		return nil, ErrInvalidURL
	}

	if localhostOnly && !isLoopbackHost(hostname) {
		return nil, ErrDeniedURL
	}

	return parsed, nil
}

func RedactForLog(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	if parsed.RawQuery != "" {
		parsed.RawQuery = "redacted"
	}
	parsed.Fragment = ""
	return parsed.String()
}

func IsLoopbackURL(parsed *url.URL) bool {
	if parsed == nil {
		return false
	}
	return isLoopbackHost(strings.ToLower(parsed.Hostname()))
}

func RewriteLoopbackURL(parsed *url.URL, localPort int) string {
	clone := *parsed
	clone.Host = net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", localPort))
	return clone.String()
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
